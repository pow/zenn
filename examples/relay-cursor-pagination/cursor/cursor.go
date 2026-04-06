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
