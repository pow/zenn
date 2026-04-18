# TypeScript API エラーコード設計 — サンプルコード

記事「TypeScript でエラーコード体系を設計して API エラーレスポンスを統一する」のサンプルコードです。

## セットアップ

```bash
npm install
```

## テスト実行

```bash
npm test
```

## ファイル構成

- `src/error-codes.ts` — エラーコード定数とユニオン型
- `src/app-error.ts` — AppError クラス
- `src/http-status.ts` — エラーコード → HTTP ステータスマッピング
- `tests/app-error.test.ts` — AppError のテスト
- `tests/http-status.test.ts` — マッピングのテスト
