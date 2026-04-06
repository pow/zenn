package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// EncodeCursor は sequence ID と UUID から Relay スタイルのカーソルを生成する。
// フォーマット: base64("TypeName:sequenceID:uuid")
func EncodeCursor(typeName string, sequenceID int, id uuid.UUID) string {
	raw := fmt.Sprintf("%s:%d:%s", typeName, sequenceID, id.String())
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor はカーソル文字列を sequence ID と UUID にデコードする。
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
