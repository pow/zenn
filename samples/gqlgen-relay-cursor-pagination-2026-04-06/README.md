# gqlgen Relay カーソルページネーション サンプルコード

記事「gqlgen で Relay スタイルのカーソルページネーションを実装する」のサンプルコードです。

## セットアップ

```bash
go mod tidy
```

## テスト実行

```bash
go test ./... -v
```

## ファイル構成

- `cursor.go` — カーソルの encode/decode 関数
- `pagination.go` — Keyset ページネーションのクエリ構築ロジック
- `cursor_test.go` — カーソル関連のテスト
- `pagination_test.go` — ページネーションクエリ構築のテスト
