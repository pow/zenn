package pagination

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// PageRequest はページネーションリクエストのパラメータを表す。
type PageRequest struct {
	First int
	After *string // カーソル（省略可）
}

// PageInfo は Relay Connection 仕様の PageInfo を表す。
type PageInfo struct {
	HasNextPage bool
	EndCursor   *string
}

// Edge はノードとカーソルのペアを表す。
type Edge[T any] struct {
	Node   T
	Cursor string
}

// Connection は Relay Connection 仕様のレスポンスを表す。
type Connection[T any] struct {
	Edges      []Edge[T]
	PageInfo   PageInfo
	TotalCount int
}

// KeysetCondition は keyset pagination の WHERE 条件を構築する。
// ORDER BY (sequence_id ASC, id ASC) を前提とし、
// カーソル位置より後のレコードを取得する条件を返す。
func KeysetCondition(afterSeqID int, afterID uuid.UUID) (clause string, args []any) {
	clause = "(sequence_id > ? OR (sequence_id = ? AND id > ?))"
	args = []any{afterSeqID, afterSeqID, afterID.String()}
	return clause, args
}

// BuildPaginatedQuery は keyset pagination 付きの SQL クエリを構築する。
// baseQuery は "SELECT ... FROM ... WHERE ..." の形式で、追加条件として
// keyset 条件と LIMIT が付加される。
func BuildPaginatedQuery(baseQuery string, req PageRequest, typeName string) (query string, args []any, err error) {
	var conditions []string

	if req.After != nil {
		seqID, id, err := DecodeCursor(typeName, *req.After)
		if err != nil {
			return "", nil, fmt.Errorf("invalid after cursor: %w", err)
		}
		clause, cursorArgs := KeysetCondition(seqID, id)
		conditions = append(conditions, clause)
		args = append(args, cursorArgs...)
	}

	query = baseQuery
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY sequence_id ASC, id ASC"

	// first + 1 を LIMIT に指定して hasNextPage を判定する
	limit := req.First + 1
	query += fmt.Sprintf(" LIMIT %d", limit)

	return query, args, nil
}
