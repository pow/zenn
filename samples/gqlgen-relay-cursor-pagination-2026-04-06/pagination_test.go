package pagination

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildPaginatedQuery_firstPage(t *testing.T) {
	req := PageRequest{First: 10, After: nil}
	query, args, err := BuildPaginatedQuery("SELECT * FROM users WHERE org_id = ?", req, "User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args for first page, got %d", len(args))
	}
	if !strings.Contains(query, "ORDER BY sequence_id ASC, id ASC") {
		t.Error("query should contain ORDER BY clause")
	}
	if !strings.Contains(query, "LIMIT 11") {
		t.Errorf("query should contain LIMIT 11 (first+1), got: %s", query)
	}
}

func TestBuildPaginatedQuery_withCursor(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	cursor := EncodeCursor("User", 5, id)
	req := PageRequest{First: 10, After: &cursor}

	query, args, err := BuildPaginatedQuery("SELECT * FROM users WHERE org_id = ?", req, "User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 3 {
		t.Errorf("expected 3 args for cursor page, got %d", len(args))
	}
	if !strings.Contains(query, "sequence_id > ?") {
		t.Error("query should contain keyset condition")
	}
	if !strings.Contains(query, "LIMIT 11") {
		t.Errorf("query should contain LIMIT 11, got: %s", query)
	}
}

func TestBuildPaginatedQuery_invalidCursor(t *testing.T) {
	badCursor := "invalid-cursor"
	req := PageRequest{First: 10, After: &badCursor}

	_, _, err := BuildPaginatedQuery("SELECT * FROM users WHERE org_id = ?", req, "User")
	if err == nil {
		t.Fatal("expected error for invalid cursor, got nil")
	}
}

func TestKeysetCondition_returnsCorrectClauseAndArgs(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	clause, args := KeysetCondition(5, id)

	if !strings.Contains(clause, "sequence_id > ?") {
		t.Errorf("clause should reference sequence_id, got: %s", clause)
	}
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
	if args[0] != 5 {
		t.Errorf("first arg should be 5, got %v", args[0])
	}
}
