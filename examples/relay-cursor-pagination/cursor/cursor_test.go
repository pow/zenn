package cursor_test

import (
	"testing"

	"github.com/example/relay-cursor-pagination/cursor"
)

func TestEncodeAndDecode(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"simple id", "user-123"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"numeric", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := cursor.Encode(tt.id)

			// カーソルは不透明であるべき（元のIDがそのまま見えない）
			if encoded == tt.id {
				t.Errorf("Encode should produce an opaque string, got same as input: %s", encoded)
			}

			decoded, err := cursor.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if decoded != tt.id {
				t.Errorf("Decode got %q, want %q", decoded, tt.id)
			}
		})
	}
}

func TestDecodeInvalidCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"not base64", "!!!invalid-base64!!!"},
		{"valid base64 but wrong prefix", "dXNlci0xMjM="}, // "user-123" without prefix
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cursor.Decode(tt.cursor)
			if err == nil {
				t.Error("Decode should return error for invalid cursor")
			}
		})
	}
}
