# Next.js Route Handler で PDF を動的に生成するサンプルコード

記事「Next.js Route Handler で PDF を動的に生成して返す方法」のサンプルコードです。

## セットアップ

```bash
npm install
```

## テスト実行

```bash
npm test
```

## ファイル構成

- `src/invoice-document.tsx` — 請求書 PDF のレイアウトコンポーネント
- `src/render-pdf.ts` — PDF レンダリングユーティリティ
- `tests/render-pdf.test.ts` — PDF 生成の検証テスト
