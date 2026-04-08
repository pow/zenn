package pagination

import (
	"encoding/base64"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	id := "user-123"
	cursor := EncodeCursor(id)

	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("DecodeCursor failed: %v", err)
	}
	if decoded != id {
		t.Errorf("got %q, want %q", decoded, id)
	}
}

func TestDecodeInvalidBase64(t *testing.T) {
	_, err := DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodeInvalidFormat(t *testing.T) {
	cursor := base64.StdEncoding.EncodeToString([]byte("wrong-prefix:123"))
	_, err := DecodeCursor(cursor)
	if err == nil {
		t.Error("expected error for invalid cursor format")
	}
}
