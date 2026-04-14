package pagination

import (
	"encoding/base64"
	"fmt"
)

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
