---
title: "gqlgen で Relay スタイルのカーソルページネーションを実装する"
emoji: "📄"
type: "tech"
topics: ["go", "graphql", "gqlgen", "pagination"]
published: false
---

## はじめに

Go × gqlgen で GraphQL API を開発していると、一覧取得にページネーションが必要になります。手軽な OFFSET ベースで実装しがちですが、データ量が増えると深刻なパフォーマンス問題に直面します。

本記事では、OFFSET の問題点を整理したうえで、Relay Connection 仕様(*1)に沿ったカーソルページネーションを keyset pagination で実装する方法を紹介します。

## OFFSET ページネーションの限界

OFFSET ベースの `SELECT * FROM users ORDER BY id LIMIT 20 OFFSET 10000` は、DB がまず 10,000 行を読み飛ばしてから 20 行を返します。ページが深くなるほどスキャン量が増え、レスポンスが遅くなります(*3)。

さらに、ページ間でデータの挿入・削除が起きると行がスキップされたり重複したりする問題もあります。

**keyset pagination**（seek method）はこれを解決します。前ページの最後のレコードの値を基準に `WHERE` で絞り込むため、常にインデックスを活用でき、データ量に依存しない一定のパフォーマンスを実現します(*3)。

## Relay Connection 仕様のスキーマ設計

GraphQL エコシステムでは、Relay の Connection 仕様(*1)がページネーションの標準パターンです。gqlgen(*2)でスキーマを定義する場合、以下のように Connection / Edge / PageInfo 型を用意します。

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
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  endCursor: String
}
```

`first` で取得件数、`after` で前ページの末尾カーソルを指定します。`PageInfo.hasNextPage` と `endCursor` を使って次のページをフェッチするシンプルな設計です。

## カーソルの encode / decode を実装する

カーソルはクライアントにとって**不透明な文字列**であるべきです。内部的には、keyset pagination に必要なソートキーの値をエンコードします。`(sequence_id, id)` の複合キーを base64 エンコードする実装例を示します。

```go
package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func EncodeCursor(typeName string, sequenceID int, id uuid.UUID) string {
	raw := fmt.Sprintf("%s:%d:%s", typeName, sequenceID, id.String())
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(typeName string, cursor string) (sequenceID int, id uuid.UUID, err error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	parts := strings.SplitN(string(data), ":", 3)
	if len(parts) != 3 || parts[0] != typeName {
		return 0, uuid.Nil, fmt.Errorf("invalid cursor format: expected type %s", typeName)
	}

	seqID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("invalid sequence id in cursor: %w", err)
	}

	uid, err := uuid.Parse(parts[2])
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("invalid uuid in cursor: %w", err)
	}

	return seqID, uid, nil
}
```

`typeName` をプレフィックスに含めることで、異なるエンティティのカーソルが誤って使われた場合にデコード時点で検出できます。なぜ `sequence_id` と `id` の複合キーにするかというと、`sequence_id` だけでは一意性が保証されないケースがあり、UUID を組み合わせることで確実に一意なカーソル位置を指定できるためです。

## Keyset ページネーションのクエリ構築

カーソルから取り出した値を使い、`WHERE` 句で keyset 条件を構築します。

```go
func KeysetCondition(afterSeqID int, afterID uuid.UUID) (clause string, args []any) {
	clause = "(sequence_id > ? OR (sequence_id = ? AND id > ?))"
	args = []any{afterSeqID, afterSeqID, afterID.String()}
	return clause, args
}
```

`(sequence_id > ?) OR (sequence_id = ? AND id > ?)` というパターンは、複合キーでのソート順を正しく再現するための定型句です。`ORDER BY sequence_id ASC, id ASC` と組み合わせることで、カーソル位置の直後のレコードから取得できます。

`hasNextPage` の判定には、`LIMIT` に `first + 1` を指定するテクニックを使います。実際に返すのは `first` 件だけですが、もう1件取れれば次ページが存在すると分かります。

```go
func BuildPaginatedQuery(baseQuery string, req PageRequest, typeName string) (query string, args []any, err error) {
	if req.After != nil {
		seqID, id, err := DecodeCursor(typeName, *req.After)
		if err != nil {
			return "", nil, fmt.Errorf("invalid after cursor: %w", err)
		}
		clause, cursorArgs := KeysetCondition(seqID, id)
		query = baseQuery + " AND " + clause
		args = append(args, cursorArgs...)
	} else {
		query = baseQuery
	}

	query += " ORDER BY sequence_id ASC, id ASC"
	query += fmt.Sprintf(" LIMIT %d", req.First+1)

	return query, args, nil
}
```

gqlgen のリゾルバーでは、このクエリで取得した結果を Connection 型に変換して返します。`first + 1` 件取れた場合は末尾の1件を除いて `edges` に詰め、`hasNextPage: true` とします。

## まとめ

- **OFFSET は大規模データで遅くなる**ため、keyset pagination でインデックスを活用した定速ページネーションを選ぶ(*3)
- **Relay Connection 仕様**(*1)に従うと、Apollo Client の `fetchMore` など既存ライブラリとの統合が容易になる
- **カーソルは base64 + 複合キー**でエンコードし、型名プレフィックスで安全にデコードする

## 参考リンク

- *1: [Relay Connection 仕様 — GraphQL Cursor Connections Specification](https://relay.dev/graphql/connections.htm)
- *2: [gqlgen 公式サイト](https://gqlgen.com/)
- *3: [Use The Index, Luke — No Offset](https://use-the-index-luke.com/no-offset)
