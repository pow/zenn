---
title: "Next.js Route Handler で PDF を動的に生成して返す方法"
emoji: "📄"
type: "tech"
topics: ["nextjs", "typescript", "react", "tips"]
published: false
---

## はじめに

Web アプリで請求書やレポートを PDF 出力したい場面は多いですが、Next.js App Router でサーバーサイド PDF 生成を行う方法はまだ情報が少ないと感じます。

本記事では、`@react-pdf/renderer` を使って React コンポーネントで PDF レイアウトを定義し、Route Handler から動的に PDF を返す実装方法を紹介します。

## @react-pdf/renderer の基本

`@react-pdf/renderer` は、React コンポーネントで PDF のレイアウトを宣言的に記述できるライブラリです(*1)。HTML の代わりに `Document`、`Page`、`View`、`Text` といった専用コンポーネントを使います。

以下は、シンプルな請求書 PDF のコンポーネント例です。

```tsx
import { Document, Page, Text, View, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  page: { padding: 40, fontSize: 12 },
  title: { fontSize: 20, marginBottom: 20 },
  row: { flexDirection: "row", justifyContent: "space-between", marginBottom: 8 },
  total: { flexDirection: "row", justifyContent: "space-between", marginTop: 16, borderTopWidth: 1, borderTopColor: "#333", paddingTop: 8, fontWeight: "bold" },
});

type Item = { name: string; amount: number };

type InvoiceProps = {
  invoiceNo: string;
  items: Item[];
};

export function InvoiceDocument({ invoiceNo, items }: InvoiceProps) {
  const total = items.reduce((sum, item) => sum + item.amount, 0);
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        <Text style={styles.title}>請求書 #{invoiceNo}</Text>
        {items.map((item, i) => (
          <View key={i} style={styles.row}>
            <Text>{item.name}</Text>
            <Text>¥{item.amount.toLocaleString()}</Text>
          </View>
        ))}
        <View style={styles.total}>
          <Text>合計</Text>
          <Text>¥{total.toLocaleString()}</Text>
        </View>
      </Page>
    </Document>
  );
}
```

ポイントは、通常の React コンポーネントと同じように props を受け取れるため、データに応じた動的な PDF を生成できることです。スタイルは `StyleSheet.create` で定義し、Flexbox ベースのレイアウトが使えます(*1)。

## Route Handler で PDF を返す

Next.js App Router の Route Handler を使えば、上記コンポーネントをサーバーサイドでレンダリングし、PDF バイナリをレスポンスとして返せます(*2)。

```tsx
import { renderToBuffer } from "@react-pdf/renderer";
import { NextResponse } from "next/server";
import { InvoiceDocument } from "@/components/invoice-document";

export async function GET(request: Request) {
  const items = [
    { name: "Web サイト制作", amount: 300000 },
    { name: "保守・運用（3ヶ月）", amount: 90000 },
  ];

  const buffer = await renderToBuffer(
    <InvoiceDocument invoiceNo="2026-001" items={items} />
  );

  return new NextResponse(buffer, {
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": 'attachment; filename="invoice.pdf"',
    },
  });
}
```

`renderToBuffer` は `@react-pdf/renderer` が提供する Node.js 向け API で、React コンポーネントを PDF バイナリ（`Buffer`）に変換します(*1)。これを `NextResponse` に渡し、適切な `Content-Type` と `Content-Disposition` ヘッダーを設定すれば、ブラウザ側で PDF ダウンロードが開始されます。

実際のアプリケーションでは、URL パラメータや DB から取得したデータを props に渡すことで、リクエストごとに異なる PDF を生成できます。

## なぜ PDF 生成はメインスレッドをロックするのか

ここまでのコードは動作しますが、**本番環境ではそのまま使うと危険**です。

Node.js はシングルスレッドのイベントループで動作します。`@react-pdf/renderer` は内部で Yoga レイアウトエンジンを使い、各要素の座標・サイズを**同期的に**計算します(*1)。`renderToBuffer` は Promise を返すため一見非同期に見えますが、レイアウト計算の本体は同期処理です。実行中はイベントループが完全に停止します。

つまり、あるユーザーの PDF 生成中は、**同じ Node.js プロセス上の他のすべてのリクエストが待ち状態になります**。ページ数が多い PDF やフォント埋め込みが重なると、数百ミリ秒〜数秒のブロックが発生し、他ユーザーへの応答遅延に直結します。

以下のコードで、`Promise.all` で2つの PDF を「並行」生成しても合計時間がほぼ変わらないことを確認できます。

```typescript
// 逐次実行
const start1 = performance.now();
await renderToBuffer(<InvoiceDocument {...props} />);
await renderToBuffer(<InvoiceDocument {...props} />);
console.log(`逐次: ${performance.now() - start1}ms`);

// Promise.all — 並行に見えるが実際はブロックが連続する
const start2 = performance.now();
await Promise.all([
  renderToBuffer(<InvoiceDocument {...props} />),
  renderToBuffer(<InvoiceDocument {...props} />),
]);
console.log(`Promise.all: ${performance.now() - start2}ms`);
// → 両者の所要時間はほぼ同じ
```

### Worker Thread で別スレッドに逃がす

Node.js の `worker_threads` を使えば、PDF 生成を別スレッドで実行でき、メインスレッドのイベントループをブロックしません(*3)。

```typescript
// pdf.worker.ts — 別スレッドで PDF を生成
import { parentPort, workerData } from "node:worker_threads";
import { renderToBuffer } from "@react-pdf/renderer";
import { InvoiceDocument } from "./invoice-document";

const buffer = await renderToBuffer(
  <InvoiceDocument {...workerData} />
);
parentPort?.postMessage(buffer, [buffer.buffer]);
```

```typescript
// route.ts — メインスレッドからワーカーを起動
import { Worker } from "node:worker_threads";

function renderPdfInWorker(props: InvoiceProps): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const worker = new Worker(
      new URL("./pdf.worker.ts", import.meta.url),
      { workerData: props }
    );
    worker.on("message", (buf) => resolve(Buffer.from(buf)));
    worker.on("error", reject);
  });
}
```

Worker Thread を使えば PDF 生成中もメインスレッドは他のリクエストを処理でき、ユーザー体験の劣化を防げます。複数の同時リクエストが予想される場合は積極的に検討してください。

## スタイリングの実践 Tips

`@react-pdf/renderer` のスタイルは CSS のサブセットですが、いくつか注意点があります。

- **Flexbox が基本**: `flexDirection`、`justifyContent`、`alignItems` でレイアウトを組む
- **単位は pt**: `padding: 40` は 40pt。`px`、`%` も使用可能
- **フォント**: デフォルトでは日本語が表示されない場合がある。`Font.register` でカスタムフォント（例: Noto Sans JP）を登録することで対応可能(*1)

テーブルのような表組みは、`flexDirection: "row"` の `View` を行として並べる方法がシンプルです。上記の請求書コードの `row` スタイルがその例です。

## まとめ

- `@react-pdf/renderer` を使えば、React コンポーネントとして PDF レイアウトを宣言的に定義できる
- Next.js Route Handler と `renderToBuffer` を組み合わせることで、サーバーサイドで PDF を動的生成し、レスポンスとして返せる
- `renderToBuffer` は内部で同期的なレイアウト計算を行うため、メインスレッドをブロックする。本番では Worker Thread への分離を検討する

## 参考リンク

- *1: [React-pdf 公式ドキュメント](https://react-pdf.org/)
- *2: [Next.js Route Handlers ドキュメント](https://nextjs.org/docs/app/getting-started/route-handlers)
- *3: [Node.js worker_threads ドキュメント](https://nodejs.org/api/worker_threads.html)
