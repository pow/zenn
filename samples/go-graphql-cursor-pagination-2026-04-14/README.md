# Go GraphQL Cursor Pagination サンプル

記事「Go の GraphQL API にカーソルページネーションを後付けする」のサンプルコードです。

## セットアップ

Go 1.22 以上が必要です。

```bash
cd samples/go-graphql-cursor-pagination-2026-04-14
```

## テスト実行

```bash
go test -v ./...
```

## ファイル構成

- `pagination.go` — カーソルエンコード/デコード + 汎用 Paginate 関数
- `pagination_test.go` — 各関数のユニットテスト
