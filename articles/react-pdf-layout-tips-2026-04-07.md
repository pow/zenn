---
title: "@react-pdf/renderer でPDFのレイアウト崩れを防ぐ3つのテクニック"
emoji: "📄"
type: "tech"
topics: ["typescript", "javascript", "tips"]
published: false
---

## はじめに

「ブラウザではきれいに表示されているのに、PDF にすると崩れる」——`@react-pdf/renderer` を使っている開発者なら、一度はこの絶望を味わったことがあるのではないでしょうか。

私もまさにそうでした。レポート機能の PDF エクスポートを実装したとき、ブラウザ上のプレビューは完璧。意気揚々とPDFをダウンロードして開いたら、カードが中途半端にページをまたいで真っ二つ、積み上げバーチャートの幅はガタガタ、長いユーザー名がセルからはみ出して隣の列を侵食……。デザイナーに見せたら「これはちょっと……」の一言。つらい。

原因は、`@react-pdf/renderer` が内部で使っている **Yoga レイアウトエンジン**にあります(*3)。ブラウザの CSS レンダリングとは別物なので、`overflow: hidden` が効かなかったり、Flexbox の挙動が微妙に違ったりします。つまり「ブラウザで動いたから大丈夫」が通用しない世界です。

本記事では、こうした PDF レイアウト崩れを実際に修正した経験から、即効性のある3つのテクニックを紹介します。

## テクニック1: `wrap={false}` でページ跨ぎを防止する

まず、最も遭遇率が高い「カードがページの途中でぶった切れる」問題です。

`@react-pdf/renderer` はデフォルトで、要素がページ末尾に収まらないと自動で分割します(*1)。HTML のブラウザレンダリングではスクロールすれば済む話ですが、PDF はページという物理的な区切りがあるため、この自動分割が「見出しだけ前のページに残って、本文が次のページに行く」といった悲劇を生みます。

解決策はシンプルで、`View` コンポーネントに `wrap={false}` を付けるだけ。ページに収まらない場合、要素が丸ごと次のページに移動します。

```tsx
import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  card: {
    border: "1pt solid #e2e8f0",
    borderRadius: 4,
    padding: 12,
    marginBottom: 8,
  },
  title: {
    fontSize: 14,
    fontWeight: "bold",
    marginBottom: 6,
  },
  body: {
    fontSize: 10,
    lineHeight: 1.6,
  },
});

type CardSectionProps = {
  title: string;
  body: string;
};

const CardSection = ({ title, body }: CardSectionProps) => (
  <View wrap={false} style={styles.card}>
    <Text style={styles.title}>{title}</Text>
    <Text style={styles.body}>{body}</Text>
  </View>
);
```

ポイントは `wrap={false}` を**セクション単位の View** に付けることです。最上位の `Page` に付けるとすべてが1ページに押し込まれてしまうため、分割を防ぎたい「かたまり」ごとに指定します(*1)。

カードが多い場合は、リストの各アイテムに適用するだけで十分です。

```tsx
const CardList = ({ items }: { items: CardSectionProps[] }) => (
  <View>
    {items.map((item, i) => (
      <CardSection key={i} title={item.title} body={item.body} />
    ))}
  </View>
);
```

## テクニック2: `flexBasis` でセグメント幅を正規化する

次に厄介なのが「データによって幅が変わるバーチャートやカラムレイアウト」です。

ブラウザの CSS なら `width: 30%` と書けば済む場面でも、`@react-pdf/renderer` では `flexGrow` だけに頼ると値が 0 のセグメントでも謎の最小幅が確保されてしまい、「30:50:20 のはずが 35:45:20 になってる……？」という微妙なズレが発生します。

ここで活躍するのが `flexBasis` です。パーセンテージで指定し、`flexGrow: 0` と `flexShrink: 0` を組み合わせることで、値の比率どおりの幅を正確に再現できます(*2)。

```tsx
import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  container: {
    flexDirection: "row",
    height: 24,
    borderRadius: 4,
    overflow: "hidden",
  },
  segment: {
    justifyContent: "center",
    alignItems: "center",
  },
  label: {
    fontSize: 8,
    color: "#ffffff",
  },
});

type Segment = {
  label: string;
  value: number;
  color: string;
};

const SegmentBar = ({ segments }: { segments: Segment[] }) => {
  const total = segments.reduce((sum, s) => sum + s.value, 0);

  return (
    <View style={styles.container}>
      {segments.map((segment, i) => {
        const ratio = total > 0 ? segment.value / total : 0;
        return (
          <View
            key={i}
            style={[
              styles.segment,
              {
                flexBasis: `${(ratio * 100).toFixed(1)}%`,
                flexGrow: 0,
                flexShrink: 0,
                backgroundColor: segment.color,
              },
            ]}
          >
            <Text style={styles.label}>{segment.label}</Text>
          </View>
        );
      })}
    </View>
  );
};
```

「`width` のパーセンテージ指定じゃダメなの？」と思うかもしれません。実は Yoga レイアウトエンジンでは `width` のパーセンテージと `flexBasis` で微妙に挙動が異なり、親コンテナの `flexDirection: "row"` との組み合わせでは `flexBasis` のほうが安定します。`flexGrow: 0` + `flexShrink: 0` で「伸びも縮みもしない」と明示するのがコツです(*2)。

## テクニック3: `maxLines` と `ellipsis` でテキスト溢れを制御する

最後は「テキストがセルからはみ出す」問題。ユーザーが入力した長い文字列をテーブルセルに表示するとき、ブラウザなら `text-overflow: ellipsis` や `-webkit-line-clamp` で簡単に制御できます。しかし PDF の世界にはこれらの CSS プロパティは存在しません。

代わりに、`@react-pdf/renderer` の `Text` コンポーネントが提供する `maxLines` と `ellipsis` プロパティを使います(*1)。

```tsx
import { View, Text, StyleSheet } from "@react-pdf/renderer";

const styles = StyleSheet.create({
  clampedText: {
    fontSize: 10,
    lineHeight: 1.5,
  },
  fixedWidthCell: {
    width: 120,
    padding: 4,
  },
  heading: {
    fontSize: 12,
    fontWeight: "bold",
    marginBottom: 4,
  },
});

const ClampedText = ({ text, maxLines = 2 }: { text: string; maxLines?: number }) => (
  <Text style={styles.clampedText} maxLines={maxLines} ellipsis="…">
    {text}
  </Text>
);

const TruncatedCell = ({ label, value, width = 120 }: {
  label: string;
  value: string;
  width?: number;
}) => (
  <View style={[styles.fixedWidthCell, { width }]}>
    <Text style={styles.heading}>{label}</Text>
    <Text style={styles.clampedText} maxLines={1} ellipsis="…">
      {value}
    </Text>
  </View>
);
```

テーブルの各セルに `TruncatedCell` を使えば、どんなに長いテキストが入ってもセル幅が固定されたまま省略表示されます。`maxLines={1}` で1行に制限し、`ellipsis="…"` で省略記号を付けるのがポイントです。「名前欄が3行に膨らんで表の高さが崩壊する」という悲劇を、たった2つの prop で防げます。

## まとめ

- **`wrap={false}`** をセクション単位の View に指定し、カードやブロックのページ跨ぎを防止する(*1)
- **`flexBasis` + `flexGrow: 0` + `flexShrink: 0`** の組み合わせで、データ比率どおりのセグメント幅を正確に再現する(*2)
- **`maxLines` と `ellipsis`** で、固定幅セル内のテキスト溢れを安全に制御する(*1)

`@react-pdf/renderer` は「React で書ける」という親しみやすさの裏に、Yoga レイアウトエンジンという別世界が隠れています。「ブラウザで動いたから大丈夫」を卒業して、PDF 特有のクセを味方につけていきましょう。この3つを押さえておけば、デザイナーにPDFを見せるときの冷や汗がだいぶ減るはずです。

## 参考リンク

- *1: [react-pdf 公式ドキュメント - Page wrapping](https://react-pdf.org/advanced#page-wrapping)
- *2: [Yoga Layout - Flex Basis, Grow, and Shrink](https://www.yogalayout.dev/docs/styling/flex-basis-grow-shrink)
- *3: [react-pdf GitHub リポジトリ](https://github.com/diegomura/react-pdf)
