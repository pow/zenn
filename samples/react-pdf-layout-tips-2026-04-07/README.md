# @react-pdf/renderer レイアウトテクニック サンプルコード

記事「@react-pdf/renderer でPDFのレイアウト崩れを防ぐ3つのテクニック」のサンプルコードです。

## セットアップ

```bash
npm install
```

## テスト実行

```bash
npm test
```

## ファイル構成

- `src/card-section.tsx` — wrap={false} を使ったページ跨ぎ防止コンポーネント
- `src/segment-bar.tsx` — flex-basis による均等幅セグメントバー
- `src/text-overflow.tsx` — maxLines / ellipsis によるテキスト溢れ制御
- `tests/pdf-render.test.tsx` — 各コンポーネントの PDF レンダリングテスト
