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
