package pagination

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
