---
title: "Go × gqlgen で Relay 準拠のカーソルページネーションを実装する"
emoji: "📄"
type: "tech"
topics: ["go", "graphql", "tips"]
published: false
---

## はじめに

GraphQL API でリスト系のエンドポイントを作ると、必ずページネーションの設計が必要になります。offset ベースのページネーション（`limit` / `offset`）は実装が簡単ですが、データの追加・削除があるとページがずれる問題があります。

Relay Connection 仕様(*1)はこの課題を解決するカーソルベースのページネーション標準です。Apollo Client や Relay など主要な GraphQL クライアントが対応しており、採用するとクライアント側の実装も統一できます。

本記事では、Go の GraphQL コード生成ライブラリ gqlgen(*2)で Relay Connection 仕様に準拠したカーソルページネーションを実装する方法を紹介します。

## offset vs cursor — なぜカーソルが推奨されるか

offset 方式では「N 件目から M 件取得」と指定します。一見シンプルですが、次の問題があります。

- **ページずれ**: 1ページ目を表示中に新しいデータが挿入されると、2ページ目に遷移した際に同じレコードが重複表示される
- **パフォーマンス**: `OFFSET 10000` のようなクエリは、DB が先頭から 10000 件をスキャンしてから結果を返すため遅い

カーソル方式では「このレコードの次から M 件取得」と指定します。カーソルは各レコードを一意に識別する不透明な文字列で、データの挿入・削除に影響されません。

## gqlgen でのスキーマ定義

Relay Connection 仕様(*1)では、`Connection` → `Edge` → `Node` の3層構造と `PageInfo` 型を定義します。以下は「ユーザー一覧」を例にしたスキーマです。

```graphql
type Query {
  users(first: Int!, after: String): UserConnection!
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type UserEdge {
  cursor: String!
  node: User!
}

type PageInfo {
  hasNextPage: Boolean!
  endCursor: String
}

type User {
  id: ID!
  name: String!
  email: String!
}
```

ポイントは `first`（取得件数）と `after`（カーソル）を引数にとることです。`totalCount` は仕様上は任意ですが、UI でページ数を表示する場合に便利です。

## リゾルバの実装

スキーマを定義したら、リゾルバで実際のデータ取得ロジックを書きます。カーソルのエンコード/デコードと SQL クエリの組み立てがポイントです。

### カーソルのエンコード/デコード

カーソルはクライアントにとって不透明（opaque）な文字列であるべきです(*1)。内部的には ID を base64 エンコードするのがシンプルな方法です。

```go
package cursor

import (
	"encoding/base64"
	"fmt"
)

const prefix = "cursor:"

// Encode はレコード ID をカーソル文字列に変換する
func Encode(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(prefix + id))
}

// Decode はカーソル文字列からレコード ID を復元する
func Decode(c string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	s := string(b)
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid cursor format")
	}
	return s[len(prefix):], nil
}
```

`prefix` を付けることで、生の ID がそのまま渡された場合を検出できます。

### リゾルバ本体

gqlgen が生成するリゾルバのシグネチャに合わせて、Connection を組み立てます。

```go
func (r *queryResolver) Users(
	ctx context.Context, first int, after *string,
) (*model.UserConnection, error) {
	// 1. カーソルがあればデコード
	var afterID string
	if after != nil {
		id, err := cursor.Decode(*after)
		if err != nil {
			return nil, err
		}
		afterID = id
	}

	// 2. first + 1 件取得して hasNextPage を判定
	limit := first + 1
	query := r.DB.NewSelect().
		Model((*entity.User)(nil)).
		OrderExpr("id ASC").
		Limit(limit)

	if afterID != "" {
		query = query.Where("id > ?", afterID)
	}

	var users []entity.User
	if err := query.Scan(ctx, &users); err != nil {
		return nil, err
	}

	// 3. hasNextPage の判定
	hasNext := len(users) > first
	if hasNext {
		users = users[:first] // 余分な1件を切り落とす
	}

	// 4. Edge と PageInfo を組み立て
	edges := make([]*model.UserEdge, len(users))
	for i, u := range users {
		edges[i] = &model.UserEdge{
			Cursor: cursor.Encode(u.ID),
			Node:   toUserModel(&u),
		}
	}

	var endCursor *string
	if len(edges) > 0 {
		endCursor = &edges[len(edges)-1].Cursor
	}

	// 5. totalCount（必要に応じて）
	total, _ := r.DB.NewSelect().
		Model((*entity.User)(nil)).
		Count(ctx)

	return &model.UserConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasNextPage: hasNext,
			EndCursor:   endCursor,
		},
		TotalCount: total,
	}, nil
}
```

`first + 1` 件を取得して、結果が `first` を超えていれば次ページありと判定するテクニックがポイントです。余分なクエリを発行せずに `hasNextPage` を判定できます。

## まとめ

- **Relay Connection 仕様**(*1)に従うと、Connection/Edge/PageInfo の標準構造でクライアント側の実装を統一できる
- **カーソルは base64 エンコード**した不透明文字列にし、内部構造をクライアントに露出しない
- **first + 1 件取得**で `hasNextPage` を効率的に判定し、COUNT クエリを避けられる

gqlgen(*2)はスキーマファーストのコード生成なので、まずスキーマを正しく定義すれば、リゾルバの実装はシンプルに保てます。

## 参考リンク

- *1: [Relay Connection 仕様 — GraphQL Cursor Connections Specification](https://relay.dev/graphql/connections.htm)
- *2: [gqlgen — Go で GraphQL サーバーを構築するライブラリ](https://gqlgen.com/)
