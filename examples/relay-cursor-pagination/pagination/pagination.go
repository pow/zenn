package pagination

import (
	"github.com/example/relay-cursor-pagination/cursor"
)

// Node はページネーション対象のノードを表すインターフェース
type Node interface {
	GetID() string
}

// Edge はカーソル付きノードを表す
type Edge[T Node] struct {
	Cursor string
	Node   T
}

// PageInfo はページネーション情報を表す
type PageInfo struct {
	HasNextPage bool
	EndCursor   *string
}

// Connection は Relay Connection 仕様に準拠したレスポンスを表す
type Connection[T Node] struct {
	Edges      []Edge[T]
	PageInfo   PageInfo
	TotalCount int
}

// Build は取得済みのノードスライスから Connection を組み立てる。
// nodes には first+1 件分のデータを渡すこと。
// first を超える件数があれば hasNextPage=true と判定し、余分な1件を切り落とす。
func Build[T Node](nodes []T, first int, totalCount int) Connection[T] {
	hasNext := len(nodes) > first
	if hasNext {
		nodes = nodes[:first]
	}

	edges := make([]Edge[T], len(nodes))
	for i, n := range nodes {
		edges[i] = Edge[T]{
			Cursor: cursor.Encode(n.GetID()),
			Node:   n,
		}
	}

	var endCursor *string
	if len(edges) > 0 {
		endCursor = &edges[len(edges)-1].Cursor
	}

	return Connection[T]{
		Edges: edges,
		PageInfo: PageInfo{
			HasNextPage: hasNext,
			EndCursor:   endCursor,
		},
		TotalCount: totalCount,
	}
}
