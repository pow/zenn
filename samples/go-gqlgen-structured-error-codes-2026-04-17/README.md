# Go × gqlgen 構造化エラーコード サンプル

記事「Go × gqlgen で構造化エラーコードを設計し GraphQL に安全に返すパターン」のサンプルコード。

## セットアップ

Go 1.22 以上が必要です。

```bash
cd samples/go-gqlgen-structured-error-codes-2026-04-17
```

## テスト実行

```bash
go test ./...
```

## ディレクトリ構成

```
apperror/
├── error.go       # AppError 型、Sentinel エラー、Extract 関数
└── error_test.go  # 各メソッドのテスト
```
