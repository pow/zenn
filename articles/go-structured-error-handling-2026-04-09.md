---
title: "Go の構造化エラーハンドリング — コード体系で運用時のデバッグを高速化する"
emoji: "🏷"
type: "tech"
topics: ["go", "tips", "architecture"]
published: false
---

## はじめに

Go の `error` はただの文字列インターフェースなので、自由に返せる反面、運用中に「このエラーは何？」「原因はどこ？」を素早く特定するのが難しくなりがちです。

私が関わっているプロジェクトでも、最初は `fmt.Errorf` で場当たり的にエラーを返していました。しかしサービスが増えるにつれ、ログからエラーを追跡する時間が無視できなくなり、**エラーコード体系**を導入しました。

本記事では、Go プロジェクトで実際に運用している**3層構造のエラー型**（Code / Message / Detail）を紹介します。標準ライブラリだけで実装でき、`errors.Is` によるマッチングも維持できます。

## 3層エラーモデルの設計

エラー情報を以下の3層に分離します(*1)。

| 層 | 役割 | クライアントに返す | 例 |
|----|------|:---:|-----|
| **Code** | プログラムで判定する固定識別子 | ✅ | `"E0003"` |
| **Message** | ユーザー向けの固定メッセージ | ✅ | `"リソースが見つかりません"` |
| **Detail** | 呼び出し元が付与する文脈情報 | ✅ | `"user 42 not found"` |
| *(cause)* | 内部エラーチェーン（ログ専用） | ❌ | `"sql: no rows"` |

ポイントは **Message を固定にする**ことです。Code ごとにメッセージを1箇所で定義し、呼び出し元から上書きさせません。これにより、生のエラー文字列がクライアントに漏れる事故を防げます。文脈は Detail に入れます。

### AppError 型の実装

```go
package apperror

import (
	"errors"
	"fmt"
	"log/slog"
)

type AppError struct {
	Code    string
	Message string
	Detail  string
	cause   error // unexported: ログ専用、クライアントには返さない
}

// 共通エラー定義
var (
	ErrUnauthorized = New("E0001", "認証が必要です")
	ErrForbidden    = New("E0002", "権限が不足しています")
	ErrNotFound     = New("E0003", "リソースが見つかりません")
	ErrConflict     = New("E0004", "データが競合しています。再度お試しください")
	ErrInvalidInput = New("E0005", "入力内容が不正です")
	ErrInternal     = New("E0006", "サーバーエラーが発生しました")
)

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}
```

`cause` フィールドを **unexported** にしているのが重要です。`json.Marshal` やテンプレート出力で意図せず生エラーが外部に出ることを構造的に防げます。

## Wrap / WithDetail でエラーを組み立てる

エラーを返す側では、3つのメソッドを使い分けます。

```go
// Wrap: 内部エラーを包む（ログで原因を追える）
func (e *AppError) Wrap(cause error) *AppError {
	return &AppError{Code: e.Code, Message: e.Message, Detail: e.Detail, cause: cause}
}

// WithDetail: 文脈情報を付与する（Message は変えない）
func (e *AppError) WithDetail(detail string) *AppError {
	return &AppError{Code: e.Code, Message: e.Message, Detail: detail, cause: e.cause}
}

// Wrapf: Wrap + Detail を同時に設定
func (e *AppError) Wrapf(cause error, format string, args ...any) *AppError {
	return &AppError{
		Code: e.Code, Message: e.Message,
		Detail: fmt.Sprintf(format, args...), cause: cause,
	}
}
```

使い方はシンプルです。ユースケース層で生エラーをラップし、意味のある `AppError` に変換します。

```go
user, err := repo.FindByID(ctx, id)
if err != nil {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound.WithDetail("user not found")
	}
	return nil, apperror.ErrInternal.Wrap(err)
}
```

## errors.Is 対応 — コードベースのマッチング

Go 1.13 以降、`errors.Is` でエラーチェーンを走査できます(*1)。`AppError` では **Code が同じなら一致**と判定するように `Is` メソッドを実装します。

```go
func (e *AppError) Error() string {
	s := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Detail != "" { s += ": " + e.Detail }
	if e.cause != nil { s += ": " + e.cause.Error() }
	return s
}

func (e *AppError) Unwrap() error { return e.cause }

func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}
```

これにより、`Wrap` や `fmt.Errorf("%w", ...)` で何重にラップされていても、元のエラーコードで判定できます。

```go
original := apperror.ErrForbidden.WithDetail("access denied")
wrapped := fmt.Errorf("usecase failed: %w", original)

errors.Is(wrapped, apperror.ErrForbidden) // true
```

## HTTP レスポンスへの安全な変換

API のハンドラ層では、`Extract` 関数でエラーチェーンから `AppError` を取り出し、JSON レスポンスに変換します。未知のエラーは自動的に `ErrInternal` になり、**生エラーが漏洩するリスクをゼロ**にします。

```go
func Extract(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal.Wrap(err) // 未知エラー → Internal に変換
}

func WriteError(w http.ResponseWriter, err error) {
	appErr := Extract(err)

	// サーバーログには完全な情報を出力
	slog.Error("request error", slog.Any("error", appErr))

	// クライアントには安全な情報のみ
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codeToStatus(appErr.Code))
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Detail:  appErr.Detail,
	})
}
```

`slog.Any("error", appErr)` で構造化ログに出力するために、`slog.LogValuer` を実装しておくと便利です(*2)。

```go
func (e *AppError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("code", e.Code),
		slog.String("message", e.Message),
	}
	if e.Detail != "" {
		attrs = append(attrs, slog.String("detail", e.Detail))
	}
	if e.cause != nil {
		attrs = append(attrs, slog.String("cause", e.cause.Error()))
	}
	return slog.GroupValue(attrs...)
}
```

ログ出力はこうなります。Code で grep でき、cause で根本原因まで追えます。

```
ERROR request error error.code=E0003 error.message=リソースが見つかりません error.detail="user 42 not found"
```

## まとめ

- **Code + Message + Detail** の3層でクライアント表示とサーバーログを分離する。Message は固定で上書き不可にし、生エラーの漏洩を防ぐ
- **`errors.Is` をコードベースで実装**し、何重にラップされてもエラー種別を判定可能にする
- **`Extract` でプレゼンター層を守る**。未知のエラーは自動的に Internal に変換し、情報漏洩リスクをゼロにする

すべてのコードは標準ライブラリのみで実装できます。サンプルコードとテストは [samples/go-structured-error-handling-2026-04-09](https://github.com/pow/zenn/tree/main/samples/go-structured-error-handling-2026-04-09) に置いてあります。

## 参考リンク

- *1: [Working with Errors in Go 1.13 — The Go Blog](https://go.dev/blog/go1.13-errors)
- *2: [log/slog パッケージ — Go 標準ライブラリ](https://pkg.go.dev/log/slog)
