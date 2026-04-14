---
title: "Go の GraphQL API にカーソルページネーションを後付けする"
emoji: "📄"
type: "tech"
topics: ["go", "graphql", "gqlgen", "tips"]
published: false
---

## はじめに

Go + gqlgen で GraphQL API を運用していると、最初は全件返していたネストフィールドがデータ増加とともに問題になります。100件以上のレコードが1つのフィールドにぶら下がっていると、レスポンスサイズが膨らみ、クライアント側のレンダリングも遅くなります。

本記事では、**既存フィールドの後方互換性を保ちながら**カーソルベースページネーションを導入するパターンを Go のコードで紹介します。

## なぜオフセットではなくカーソルか

ページネーションには**オフセット方式**と**カーソル方式**の2つがあります。

```graphql
# オフセット方式
items(offset: 10, limit: 5)

# カーソル方式
items(first: 5, after: "opaque-cursor")
```

オフセット方式は実装が簡単ですが、ページ遷移中にデータが追加・削除されるとページ境界がずれて重複や欠落が起きます。カーソル方式は「この要素の次から」という基準点を使うため、途中のデータ変動に影響されません(*1)。

GraphQL コミュニティでは Relay の Cursor Connections Specification(*1) がデファクトスタンダードになっており、`first` / `after` / `edges` / `pageInfo` という共通インターフェースが定義されています。

## カーソルのエンコードとデコード

カーソルはクライアントにとって**opaque（不透明）**であるべきです(*1)。内部的には ID をエンコードしているだけですが、クライアントがカーソルの中身に依存するのを防ぐため base64 でラップします。

```go
package pagination

import (
	"encoding/base64"
	"fmt"
)

// EncodeCursor は ID を opaque なカーソル文字列に変換する。
func EncodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString([]byte("cursor:" + id))
}

// DecodeCursor はカーソル文字列を元の ID に復元する。
func DecodeCursor(cursor string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	s := string(b)
	const prefix = "cursor:"
	if len(s) < len(prefix) || s[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid cursor format")
	}
	return s[len(prefix):], nil
}
```

`cursor:` プレフィックスを付けることで、任意の base64 文字列を誤ってデコードしてしまうのを防ぎます。

## ジェネリクスで汎用 Paginate 関数を作る

Go 1.18 以降のジェネリクスを使って、任意の型に対応する `Paginate` 関数を作ります(*2)。

```go
// Edge はページネーション結果の1要素。
type Edge[T any] struct {
	Node   T
	Cursor string
}

// PageInfo はページネーションのメタ情報。
type PageInfo struct {
	HasNextPage bool
	EndCursor   *string
}

// Connection はページネーション結果全体。
type Connection[T any] struct {
	Edges    []Edge[T]
	PageInfo PageInfo
}

// Paginate はソート済みスライスにカーソルベースページネーションを適用する。
// first が nil なら全件返す（後方互換）。
func Paginate[T any](
	items []T,
	getID func(T) string,
	first *int,
	after *string,
) (*Connection[T], error) {
	startIndex := 0
	if after != nil {
		afterID, err := DecodeCursor(*after)
		if err != nil {
			return nil, err
		}
		found := false
		for i, item := range items {
			if getID(item) == afterID {
				startIndex = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("cursor not found")
		}
	}

	remaining := items[startIndex:]
	limit := len(remaining)
	hasNextPage := false
	if first != nil {
		if *first < 0 {
			return nil, fmt.Errorf("first must be non-negative")
		}
		if *first < limit {
			limit = *first
			hasNextPage = true
		}
	}

	sliced := remaining[:limit]
	edges := make([]Edge[T], len(sliced))
	for i, item := range sliced {
		edges[i] = Edge[T]{
			Node:   item,
			Cursor: EncodeCursor(getID(item)),
		}
	}

	var endCursor *string
	if len(edges) > 0 {
		ec := edges[len(edges)-1].Cursor
		endCursor = &ec
	}

	return &Connection[T]{
		Edges:    edges,
		PageInfo: PageInfo{HasNextPage: hasNextPage, EndCursor: endCursor},
	}, nil
}
```

### 設計のポイント: `first` が nil なら全件返す

既存のフィールドにページネーションを後付けする場合、**`first` 引数なしの既存クエリが壊れないこと**が最も重要です。`first` を省略すると（= `nil`）全件返すようにしておけば、クライアント側を段階的に移行できます。

新しいクライアントは `first: 10, after: "..."` でページネーションし、既存クライアントはそのまま全件取得を続けられます。

### gqlgen のスキーマ例

gqlgen のスキーマに `first` / `after` 引数を追加し、リゾルバーから `Paginate` を呼び出すだけで導入できます。

```graphql
type Query {
  items(first: Int, after: String): ItemConnection!
}

type ItemConnection {
  edges: [ItemEdge!]!
  pageInfo: PageInfo!
}
```

`first` が `Int`（nullable）であることで、引数なしのクエリが引き続き動作します。

## まとめ

- **カーソルベース**はデータ変動に強く、GraphQL では Relay Cursor Connections Specification(*1) に従うのが標準
- **`first` を nullable** にして nil なら全件返す設計で、既存クエリを壊さずにページネーションを後付けできる
- Go のジェネリクス(*2)を使えば型ごとにページネーション関数を書く必要がなく、1つの `Paginate[T]` で済む

## 参考リンク

- *1: [Relay Cursor Connections Specification](https://relay.dev/graphql/connections.htm)
- *2: [Tutorial: Getting started with generics - The Go Programming Language](https://go.dev/doc/tutorial/generics)
