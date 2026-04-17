---
title: "Go × gqlgen で構造化エラーコードを設計し GraphQL に安全に返すパターン"
emoji: "🏷"
type: "tech"
topics: ["go", "graphql", "gqlgen", "tips"]
published: false
---

## はじめに

Go + gqlgen で GraphQL API を開発していると、エラーの返し方がサービスごとに散らかりがちです。`fmt.Errorf` で返した内部エラーがそのままクライアントに漏れたり、エラーコードの採番ルールがなくて同じコードが別サービスで重複したり、といった問題が起きます。

本記事では、**AppError 型 + Extract 関数 + ErrorPresenter** の3層でエラーを整理するパターンを紹介します。マイクロサービスで実際に運用して効果があった設計を、一般化した形でまとめます。

## 素朴な実装の問題点

まず、よくある素朴な実装を見てみます。

```go
// ❌ リゾルバーで直接エラーメッセージを組み立てている
func (r *queryResolver) User(ctx context.Context, id string) (*User, error) {
    user, err := r.userRepo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return user, nil
}
```

この実装には3つの問題があります。

1. **内部エラーの漏洩**: `%w` で wrap した DB エラーの詳細がクライアントに返る
2. **分類の不統一**: 「not found」なのか「forbidden」なのかが文字列でしか判別できない
3. **コードの重複**: 複数サービスで独自にエラーを定義すると、フロントエンドが各サービスのエラーを個別にハンドリングする必要がある

## 構造化エラー型を設計する

解決策は、**Code / Message / Detail / cause** の4層構造を持つエラー型を定義することです(*1)。

```go
package apperror

import (
    "errors"
    "fmt"
)

// AppError は構造化されたアプリケーションエラー。
// cause はサーバーサイドのログ用で、クライアントには返さない。
type AppError struct {
    Code    string // "E0001" のようなエラーコード
    Message string // ユーザー向けメッセージ（カテゴリ名）
    Detail  string // コンテキスト固有の詳細（空の場合あり）
    cause   error  // 内部原因（ログ用、クライアントには非���開）
}

func New(code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

func (e *AppError) Error() string {
    s := e.Code + ": " + e.Message
    if e.Detail != "" {
        s += ": " + e.Detail
    }
    if e.cause != nil {
        s += ": " + e.cause.Error()
    }
    return s
}

func (e *AppError) Unwrap() error { return e.cause }

// Is はコードが一致すれば同じエラーと判定する。
// errors.Is(err, ErrNotFound) のように使える。
func (e *AppError) Is(target error) bool {
    t, ok := target.(*AppError)
    if !ok {
        return false
    }
    return e.Code == t.Code
}
```

### Wrap と WithDetail で原因を保持する

`Wrap` は内部エラーをチェーンしつつ、同じコードとメッセージを引き継ぎます。`WithDetail` はクライアントに返す詳細情報を付加します。どちらも新しい `*AppError` を返すため、元の Sentinel エラーを汚しません(*2)。

```go
// Wrap は cause を保持した新しい AppError を返す。
func (e *AppError) Wrap(cause error) *AppError {
    return &AppError{
        Code: e.Code, Message: e.Message,
        Detail: e.Detail, cause: cause,
    }
}

// WithDetail はクライアント向けの詳細を付加する。
func (e *AppError) WithDetail(detail string) *AppError {
    return &AppError{
        Code: e.Code, Message: e.Message,
        Detail: detail, cause: e.cause,
    }
}

// Wrapf は cause + フォーマット済み Detail を付加する。
func (e *AppError) Wrapf(cause error, format string, args ...any) *AppError {
    return &AppError{
        Code: e.Code, Message: e.Message,
        Detail: fmt.Sprintf(format, args...), cause: cause,
    }
}
```

### Sentinel エラーとサービス別コード範囲

共通エラーを Sentinel 変数として定義し、サービスごとにコード範囲を割り当てます。

```go
// 共通エラー（全サービス共通）
var (
    ErrUnauthorized = New("E0001", "Authentication required.")
    ErrForbidden    = New("E0002", "Permission denied.")
    ErrNotFound     = New("E0003", "Resource not found.")
    ErrConflict     = New("E0004", "Data conflict. Please retry.")
    ErrInvalidInput = New("E0005", "Invalid input.")
    ErrInternal     = New("E0006", "Internal server error.")
)
```

| 範囲 | サービス |
|------|---------|
| E0001–E0099 | 共通 |
| E0100–E0199 | User サービス |
| E0200–E0299 | Organization サービス |
| E0300–E0399 | Project サービス |

各サービスで独自のエラーを追加する場合は、割り当てられた範囲内で定義します。

```go
// user サービス固有のエラー
var ErrEmailAlreadyExists = New("E0101", "Email already registered.")
```

## Extract で安全にエラーを取り出す

Use Case 層やリゾルバーから返されたエラーチェーンの中から `AppError` を取り出す `Extract` 関数を用意します(*2)。`AppError` が見つからなければ `ErrInternal` を返すことで、**生のエラーがクライアントに漏れることを構造的に防ぎます**。

```go
// Extract はエラーチェーンから AppError を取り出す。
// 見つからなければ ErrInternal を返し、生エラーの漏洩を防ぐ。
func Extract(err error) *AppError {
    var appErr *AppError
    if errors.As(err, &appErr) {
        return appErr
    }
    return ErrInternal
}
```

Use Case 層では `Wrap` を使い、リポジトリから返った生エラーに AppError を被せます。

```go
func (uc *GetUserUseCase) Execute(ctx context.Context, id string) (*User, error) {
    user, err := uc.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, apperror.ErrNotFound.Wrapf(err, "user %s", id)
        }
        return nil, apperror.ErrInternal.Wrap(err)
    }
    return user, nil
}
```

## gqlgen の ErrorPresenter で GraphQL に変換する

最後のピースは、gqlgen の `ErrorPresenter` です(*1)。`AppError` を GraphQL の `extensions` フィールドにマッピングし、未知のエラーは `ErrInternal` としてマスクします。

```go
package graph

import (
    "context"
    "errors"
    "log/slog"

    "github.com/99designs/gqlgen/graphql"
    "github.com/vektah/gqlparser/v2/gqlerror"
)

func ErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
    err := graphql.DefaultErrorPresenter(ctx, e)
    if err == nil {
        return nil
    }

    var appErr *apperror.AppError
    if errors.As(e, &appErr) {
        slog.WarnContext(ctx, "GraphQL error", slog.Any("error", appErr))
        err.Message = appErr.Message
        err.Extensions = map[string]any{
            "code":    appErr.Code,
            "message": appErr.Message,
            "detail":  appErr.Detail,
        }
    } else {
        slog.ErrorContext(ctx, "unexpected error", slog.String("error", e.Error()))
        err.Message = apperror.ErrInternal.Message
        err.Extensions = map[string]any{
            "code":    apperror.ErrInternal.Code,
            "message": apperror.ErrInternal.Message,
            "detail":  "",
        }
    }
    return err
}
```

gqlgen のサーバー初期化時に `SetErrorPresenter` で登録します。

```go
srv := handler.New(generated.NewExecutableSchema(cfg))
srv.SetErrorPresenter(graph.ErrorPresenter)
```

これにより、クライアントが受け取る GraphQL レスポンスは以下の形になります。

```json
{
  "errors": [{
    "message": "Resource not found.",
    "extensions": {
      "code": "E0003",
      "message": "Resource not found.",
      "detail": "user abc-123"
    }
  }]
}
```

フロントエンドは `extensions.code` で分岐できるため、エラーメッセージの文言変更に影響されません(*3)。

## まとめ

- **AppError 型**で Code / Message / Detail / cause を分離し、`cause` はサーバーログ専用にして内部エラーの漏洩を防ぐ(*2)
- **Extract 関数**で `errors.As` を使って AppError を取り出し、見つからなければ `ErrInternal` にフォールバックする安全設計にする
- **ErrorPresenter** で AppError を GraphQL `extensions` に変換し、フロントエンドがコードベースでエラーハンドリングできるようにする(*1)

サービスごとにエラーコード範囲を割り当てておくと、マイクロサービスが増えてもコード衝突を防げます。

## 参考リンク

- *1: [Handling Errors — gqlgen](https://gqlgen.com/reference/errors/)
- *2: [Package errors — Go 標準ライブラリ](https://pkg.go.dev/errors)
- *3: [GraphQL Specification — Errors](https://spec.graphql.org/October2021/#sec-Errors)
