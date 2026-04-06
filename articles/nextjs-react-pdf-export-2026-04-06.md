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

## スタイリングの実践 Tips

`@react-pdf/renderer` のスタイルは CSS のサブセットですが、いくつか注意点があります。

- **Flexbox が基本**: `flexDirection`、`justifyContent`、`alignItems` でレイアウトを組む
- **単位は pt**: `padding: 40` は 40pt。`px`、`%` も使用可能
- **フォント**: デフォルトでは日本語が表示されない場合がある。`Font.register` でカスタムフォント（例: Noto Sans JP）を登録することで対応可能(*1)

テーブルのような表組みは、`flexDirection: "row"` の `View` を行として並べる方法がシンプルです。上記の請求書コードの `row` スタイルがその例です。

## まとめ

- `@react-pdf/renderer` を使えば、React コンポーネントとして PDF レイアウトを宣言的に定義できる
- Next.js Route Handler と `renderToBuffer` を組み合わせることで、サーバーサイドで PDF を動的生成し、レスポンスとして返せる
- スタイルは Flexbox ベースの CSS サブセットで記述し、日本語フォントは `Font.register` で対応する

## 参考リンク

- *1: [React-pdf 公式ドキュメント](https://react-pdf.org/)
- *2: [Next.js Route Handlers ドキュメント](https://nextjs.org/docs/app/getting-started/route-handlers)
