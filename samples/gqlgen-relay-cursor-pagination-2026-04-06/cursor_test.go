package pagination

import (
	"testing"

	"github.com/google/uuid"
)

func TestEncodeDecode_roundTrip(t *testing.T) {
	typeName := "User"
	seqID := 42
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	cursor := EncodeCursor(typeName, seqID, id)

	gotSeq, gotID, err := DecodeCursor(typeName, cursor)
	if err != nil {
		t.Fatalf("DecodeCursor returned error: %v", err)
	}
	if gotSeq != seqID {
		t.Errorf("sequence ID = %d, want %d", gotSeq, seqID)
	}
	if gotID != id {
		t.Errorf("UUID = %s, want %s", gotID, id)
	}
}

func TestDecodeCursor_invalidBase64(t *testing.T) {
	_, _, err := DecodeCursor("User", "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestDecodeCursor_wrongTypeName(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	cursor := EncodeCursor("User", 1, id)

	_, _, err := DecodeCursor("Post", cursor)
	if err == nil {
		t.Fatal("expected error for wrong type name, got nil")
	}
}

func TestDecodeCursor_invalidSequenceID(t *testing.T) {
	// 手動で不正なカーソルを作成（sequence_id が数値でない）
	_, _, err := DecodeCursor("User", "VXNlcjpub3RhbnVtYmVyOjU1MGU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDAwMA==")
	if err == nil {
		t.Fatal("expected error for invalid sequence id, got nil")
	}
}
