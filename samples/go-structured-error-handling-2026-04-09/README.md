# Go 構造化エラーハンドリング サンプルコード

記事「Go の構造化エラーハンドリング — コード体系で運用時のデバッグを高速化する」のサンプルコードです。

## 前提条件

- Go 1.22 以上

## ファイル構成

```
.
├── apperror.go       # AppError 型定義・コンストラクタ・Is/Unwrap 実装
├── presenter.go      # HTTP レスポンス変換（WriteError）
├── apperror_test.go  # テスト
├── go.mod
└── README.md
```

## テスト実行

```bash
go test -v ./...
```
