---
title: "gqlgen の Error Presenter で GraphQL エラーレスポンスを構造化する"
emoji: "⚠"
type: "tech"
topics: ["go", "graphql", "gqlgen", "tips"]
published: false
---

## はじめに

Go + gqlgen で GraphQL API を開発していると、エラーレスポンスが散らかりがちです。各リゾルバーで `fmt.Errorf` や `errors.New` を使って直接メッセージを返しているうちに、クライアント側でエラーの種類を判別できない「ただの文字列エラー」が量産されていきます。

本記事では、**カスタムエラー型**と **gqlgen の Error Presenter** を組み合わせて、GraphQL 仕様に準拠した構造化エラーレスポンスを返すパターンを紹介します。

## GraphQL のエラーレスポンス仕様

GraphQL の仕様(*1)では、エラーは `errors` 配列で返されます。各エラーオブジェクトには `message` と `path` に加えて、任意の追加情報を格納する `extensions` フィールドがあります。

```json
{
  "errors": [
    {
      "message": "ユーザーが見つかりません",
      "path": ["user"],
      "extensions": {
        "code": "NOT_FOUND"
      }
    }
  ]
}
```

`extensions.code` を使えば、クライアントは文字列比較ではなくコードベースでエラーハンドリングを分岐できます。

## ステップ1: カスタムエラー型を定義する

アプリケーション内で使うエラーコードとカスタムエラー型を定義します。

```go
package apperror

import "fmt"

// Code はアプリケーションエラーの種別を表す。
type Code string

const (
    CodeNotFound   Code = "NOT_FOUND"
    CodeForbidden  Code = "FORBIDDEN"
    CodeBadRequest Code = "BAD_REQUEST"
    CodeInternal   Code = "INTERNAL"
)

// AppError はアプリケーション層のエラーを表す構造体。
type AppError struct {
    Code    Code
    Message string
    Err     error // ラップ元のエラー（ログ用）
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewNotFound(msg string) *AppError {
    return &AppError{Code: CodeNotFound, Message: msg}
}

func NewForbidden(msg string) *AppError {
    return &AppError{Code: CodeForbidden, Message: msg}
}
```

ポイントは `Unwrap()` を実装していることです。Go 標準の `errors.As` でこの型を判別できるため、エラーの伝播経路で何度 `fmt.Errorf("%w", err)` でラップされても、元の `AppError` を取り出せます(*1)。

## ステップ2: Error Presenter でエラーを変換する

gqlgen の Error Presenter は、リゾルバーが返したエラーを GraphQL レスポンスに変換するフック関数です(*2)。ここでカスタムエラー型を検出し、`extensions` 付きのレスポンスに変換します。

まず、エラーを分類する**純粋関数**を定義します。

```go
package presenter

import "errors"

// GraphQLError は GraphQL レスポンスのエラー表現。
type GraphQLError struct {
    Message    string
    Extensions map[string]interface{}
}

// ClassifyError はエラーを GraphQL レスポンス用に分類する。
func ClassifyError(err error) GraphQLError {
    var appErr *AppError
    if errors.As(err, &appErr) {
        return GraphQLError{
            Message:    appErr.Message,
            Extensions: map[string]interface{}{"code": string(appErr.Code)},
        }
    }
    // 未知のエラーは内部情報を漏らさない
    return GraphQLError{
        Message:    "internal server error",
        Extensions: map[string]interface{}{"code": "INTERNAL"},
    }
}
```

gqlgen への登録はサーバー初期化時に1行で済みます(*2)。

```go
srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
    result := ClassifyError(err)
    return &gqlerror.Error{
        Message:    result.Message,
        Path:       graphql.GetPath(ctx),
        Extensions: result.Extensions,
    }
})
```

## なぜ未知のエラーで内部情報を隠すのか

`ClassifyError` のデフォルトブランチで `"internal server error"` を返しているのは、セキュリティ上重要です。リゾルバーが DB のコネクションエラーや認証トークンのパースエラーを返した場合、その詳細をクライアントに見せると攻撃者にシステム内部の情報を与えてしまいます。

ログには元のエラー全文を記録し、クライアントには種別コードだけを返す。この境界を Error Presenter で一箇所に集約できるのが、このパターンの利点です。

## 分類ロジックを純粋関数にする利点

`ClassifyError` を gqlgen の `context` や `gqlerror` に依存しない純粋関数として切り出すことで、テストが容易になります。

```go
func TestClassifyErrorWithWrappedAppError(t *testing.T) {
    original := NewNotFound("item not found")
    wrapped := fmt.Errorf("repository: %w", original)

    result := ClassifyError(wrapped)

    if result.Extensions["code"] != "NOT_FOUND" {
        t.Errorf("expected NOT_FOUND, got %s", result.Extensions["code"])
    }
}
```

`fmt.Errorf("%w", ...)` で何層ラップされても `errors.As` が `AppError` を見つけてくれるため、リポジトリ層やサービス層でエラーにコンテキストを追加しても、Error Presenter の動作は変わりません。gqlgen の依存なしにユニットテストで高速にカバーできます。

## まとめ

- **カスタムエラー型**（`AppError`）でエラーコードとメッセージを構造化し、`errors.As` で判別可能にする(*1)
- **Error Presenter** でエラー→GraphQL レスポンスの変換を一箇所に集約し、`extensions.code` で種別を返す(*2)
- 分類ロジックを**純粋関数**に切り出すことで、gqlgen に依存しないユニットテストが書ける

GraphQL のエラーレスポンスは「ただの文字列」から始めてしまいがちですが、Error Presenter を設定するだけで一貫した構造化エラーを返せるようになります。

## 参考リンク

- *1: [GraphQL Specification — Errors](https://spec.graphql.org/October2021/#sec-Errors)
- *2: [Error Handling — gqlgen](https://gqlgen.com/reference/errors/)
