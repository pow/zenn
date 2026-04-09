---
title: "Go で GraphQL のカーソルベースページネーションを実装する"
emoji: "📄"
type: "tech"
topics: ["go", "graphql", "gqlgen", "tips"]
published: false
---

## はじめに

GraphQL API でリスト系フィールドを返していると、データ件数の増加に伴いレスポンスが肥大化します。数百件のレコードを一度に返すとレスポンスサイズが膨れ上がり、クライアントのパフォーマンスにも悪影響を与えます。

この問題の定番の解決策がページネーションです。本記事では、Relay Connection Specification に基づくカーソルベースページネーションを Go で実装する方法を紹介します。

## カーソルベースとオフセットベースの違い

ページネーションには主に2つの方式があります(*1)。

**オフセットベース**は SQL の `LIMIT 10 OFFSET 20` に対応する直感的な方式ですが、ページ取得中にデータが挿入・削除されると結果がずれます。また、`OFFSET` が大きくなると DB の走査コストも増加します。

**カーソルベース**は「この要素の次から N 件」と指定する方式です。特定の要素を起点にするため、データの挿入・削除の影響を受けにくく、大量データの段階的な取得に向いています。

## Relay Connection Specification の構造

GraphQL コミュニティでは、カーソルページネーションの標準仕様として Relay Connection Specification(*2) が広く採用されています。核となる型は `Connection`、`Edge`、`PageInfo` の3つです。

```graphql
type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  endCursor: String
}
```

クライアントは `first`（取得件数）と `after`（カーソル）を指定してページを取得します。

## Go でカーソル関数を実装する

カーソルはクライアントに内部構造を見せないよう Base64 でエンコードします。プレフィックスを付けることでデコード時のバリデーションも兼ねられます。

```go
package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const cursorPrefix = "cursor:"

func EncodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(cursorPrefix + id),
	)
}

func DecodeCursor(cursor string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	s := string(decoded)
	if !strings.HasPrefix(s, cursorPrefix) {
		return "", fmt.Errorf("invalid cursor format")
	}
	return strings.TrimPrefix(s, cursorPrefix), nil
}
```

`EncodeCursor("user-42")` は `"Y3Vyc29yOnVzZXItNDI="` のような不透明な文字列を返します。クライアントはこの値をそのまま `after` 引数に渡すだけで、内部の ID 体系を知る必要がありません。

## ページネーション関数の実装

Go のジェネリクスを活用し、任意の型に適用できるページネーション関数を作ります。

```go
type PageInfo struct {
	HasNextPage bool
	EndCursor   *string
}

type Edge[T any] struct {
	Node   T
	Cursor string
}

type Connection[T any] struct {
	Edges    []Edge[T]
	PageInfo PageInfo
}

func Paginate[T any](
	items []T, first int, after *string,
	idFunc func(T) string,
) (*Connection[T], error) {
	start := 0
	if after != nil {
		afterID, err := DecodeCursor(*after)
		if err != nil {
			return nil, err
		}
		for i, item := range items {
			if idFunc(item) == afterID {
				start = i + 1
				break
			}
		}
	}

	end := start + first
	if end > len(items) {
		end = len(items)
	}

	edges := make([]Edge[T], 0, end-start)
	for _, item := range items[start:end] {
		edges = append(edges, Edge[T]{
			Node:   item,
			Cursor: EncodeCursor(idFunc(item)),
		})
	}

	var endCursor *string
	if len(edges) > 0 {
		ec := edges[len(edges)-1].Cursor
		endCursor = &ec
	}

	return &Connection[T]{
		Edges: edges,
		PageInfo: PageInfo{
			HasNextPage: end < len(items),
			EndCursor:   endCursor,
		},
	}, nil
}
```

`idFunc` でノードから一意な識別子を取り出す関数を注入します。gqlgen のリゾルバからは、DB クエリ結果のスライスをこの関数に渡すだけでページネーションを適用できます(*3)。

## まとめ

- カーソルベースはデータ変動に強く、大量データの安全な返却に適する
- Base64 エンコードでカーソルの内部構造をクライアントから隠蔽する
- Go のジェネリクスで汎用ページネーション関数を作れば、複数リゾルバで再利用できる

## 参考リンク

- *1: [GraphQL 公式ドキュメント - Pagination](https://graphql.org/learn/pagination/)
- *2: [Relay Cursor Connections Specification](https://relay.dev/graphql/connections.htm)
- *3: [gqlgen 公式サイト](https://gqlgen.com/)
