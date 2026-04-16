# go-gqlgen-error-presenter サンプルコード

記事「gqlgen の Error Presenter で GraphQL エラーレスポンスを構造化する」のサンプルコードです。

## セットアップ

Go 1.22 以上が必要です。外部依存はありません（標準ライブラリのみ）。

## ファイル構成

- `apperror.go` — カスタムエラー型 `AppError` と `Code` の定義
- `presenter.go` — エラー分類関数 `ClassifyError`
- `presenter_test.go` — 全コード例に対応するテスト

## テスト実行

```bash
go test -v ./...
```
