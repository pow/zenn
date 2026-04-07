---
title: "@react-pdf/renderer でPDFのレイアウト崩れを防ぐ3つのテクニック"
emoji: "📄"
type: "tech"
topics: ["typescript", "javascript", "tips"]
published: false
---

## はじめに

React で PDF を生成できる `@react-pdf/renderer` は便利なライブラリですが、ブラウザの CSS とは挙動が異なるため、レイアウト崩れに悩まされることがあります。特に、データ量が可変のレポートや帳票を出力するプロジェクトでは「カードがページの途中で切れる」「グラフの幅が揃わない」「長いテキストがはみ出す」といった問題に直面しがちです。

本記事では、実際のプロジェクトで PDF レイアウト崩れを修正した経験をもとに、すぐに使える3つのテクニックを紹介します。

## テクニック1: `wrap={false}` でページ跨ぎを防止する

`@react-pdf/renderer` はデフォルトで、要素がページ末尾に収まらない場合に自動で分割します(*1)。レポートのカードやセクションが途中で切れてしまう原因はこの挙動です。

`View` コンポーネントに `wrap={false}` を指定すると、その要素が分割されず、ページに収まらない場合は次のページに丸ごと移動します。

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

横並びのバーチャートやカラムレイアウトで、データの値に応じてセグメント幅を動的に設定するケースがあります。このとき、`flexGrow` だけに頼ると値が 0 のセグメントでも最小幅が確保されてしまい、意図しないレイアウトになります。

`flexBasis` にパーセンテージを指定し、`flexGrow: 0` と `flexShrink: 0` を組み合わせると、値の比率どおりの幅を正確に再現できます(*2)。

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

`flexBasis` をパーセンテージで指定する理由は、`@react-pdf/renderer` の Yoga レイアウトエンジンが `width` のパーセンテージ指定と `flexBasis` で挙動が微妙に異なるためです。`flexBasis` + `flexGrow: 0` + `flexShrink: 0` の組み合わせが最も安定して比率を再現できます(*2)。

## テクニック3: `maxLines` と `ellipsis` でテキスト溢れを制御する

PDF では、ブラウザの `text-overflow: ellipsis` や `-webkit-line-clamp` は使えません。代わりに `@react-pdf/renderer` の `Text` コンポーネントが提供する `maxLines` と `ellipsis` プロパティを使います(*1)。

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

テーブルの各セルに `TruncatedCell` を使えば、どんなに長いテキストが入ってもセル幅が固定されたまま省略表示されます。`maxLines={1}` で1行に制限し、`ellipsis="…"` で省略記号を付けるのがポイントです。

## まとめ

- **`wrap={false}`** をセクション単位の View に指定し、カードやブロックのページ跨ぎを防止する(*1)
- **`flexBasis` + `flexGrow: 0` + `flexShrink: 0`** の組み合わせで、データ比率どおりのセグメント幅を正確に再現する(*2)
- **`maxLines` と `ellipsis`** で、固定幅セル内のテキスト溢れを安全に制御する(*1)

`@react-pdf/renderer` は CSS とは異なる Yoga レイアウトエンジンを使っているため、ブラウザでの経験がそのまま通用しない場面があります。本記事のテクニックを活用して、安定した PDF レイアウトを実現してください。

## 参考リンク

- *1: [react-pdf 公式ドキュメント - Page wrapping](https://react-pdf.org/advanced#page-wrapping)
- *2: [Yoga Layout - Flex Basis, Grow, and Shrink](https://www.yogalayout.dev/docs/styling/flex-basis-grow-shrink)
- *3: [react-pdf GitHub リポジトリ](https://github.com/diegomura/react-pdf)
