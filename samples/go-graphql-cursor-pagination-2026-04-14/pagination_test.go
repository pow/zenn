package pagination

import (
	"testing"
)

type Item struct {
	ID   string
	Name string
}

func getItemID(item Item) string {
	return item.ID
}

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }

func TestEncodeDecode(t *testing.T) {
	original := "item-42"
	cursor := EncodeCursor(original)
	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("DecodeCursor failed: %v", err)
	}
	if decoded != original {
		t.Errorf("got %q, want %q", decoded, original)
	}
}

func TestDecodeInvalidCursor(t *testing.T) {
	_, err := DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeWrongPrefix(t *testing.T) {
	// Valid base64 but wrong prefix
	encoded := "aGVsbG8=" // "hello" in base64
	_, err := DecodeCursor(encoded)
	if err == nil {
		t.Fatal("expected error for wrong cursor prefix")
	}
}

func TestPaginateFirstPage(t *testing.T) {
	items := []Item{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "3", Name: "Charlie"},
		{ID: "4", Name: "Dave"},
		{ID: "5", Name: "Eve"},
	}

	conn, err := Paginate(items, getItemID, intPtr(2), nil)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 2 {
		t.Fatalf("got %d edges, want 2", len(conn.Edges))
	}
	if conn.Edges[0].Node.Name != "Alice" {
		t.Errorf("first edge name = %q, want Alice", conn.Edges[0].Node.Name)
	}
	if conn.Edges[1].Node.Name != "Bob" {
		t.Errorf("second edge name = %q, want Bob", conn.Edges[1].Node.Name)
	}
	if !conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be true")
	}
	if conn.PageInfo.EndCursor == nil {
		t.Fatal("expected EndCursor to be non-nil")
	}
}

func TestPaginateWithAfterCursor(t *testing.T) {
	items := []Item{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "3", Name: "Charlie"},
		{ID: "4", Name: "Dave"},
		{ID: "5", Name: "Eve"},
	}

	afterCursor := EncodeCursor("2")
	conn, err := Paginate(items, getItemID, intPtr(2), &afterCursor)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 2 {
		t.Fatalf("got %d edges, want 2", len(conn.Edges))
	}
	if conn.Edges[0].Node.Name != "Charlie" {
		t.Errorf("first edge name = %q, want Charlie", conn.Edges[0].Node.Name)
	}
	if conn.Edges[1].Node.Name != "Dave" {
		t.Errorf("second edge name = %q, want Dave", conn.Edges[1].Node.Name)
	}
	if !conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be true")
	}
}

func TestPaginateLastPage(t *testing.T) {
	items := []Item{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "3", Name: "Charlie"},
	}

	afterCursor := EncodeCursor("2")
	conn, err := Paginate(items, getItemID, intPtr(5), &afterCursor)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 1 {
		t.Fatalf("got %d edges, want 1", len(conn.Edges))
	}
	if conn.Edges[0].Node.Name != "Charlie" {
		t.Errorf("edge name = %q, want Charlie", conn.Edges[0].Node.Name)
	}
	if conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be false")
	}
}

func TestPaginateNilFirstReturnsAll(t *testing.T) {
	items := []Item{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "3", Name: "Charlie"},
	}

	conn, err := Paginate(items, getItemID, nil, nil)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 3 {
		t.Fatalf("got %d edges, want 3", len(conn.Edges))
	}
	if conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be false when first is nil")
	}
}

func TestPaginateNegativeFirstReturnsError(t *testing.T) {
	items := []Item{{ID: "1", Name: "Alice"}}
	_, err := Paginate(items, getItemID, intPtr(-1), nil)
	if err == nil {
		t.Fatal("expected error for negative first")
	}
}

func TestPaginateEmptySlice(t *testing.T) {
	var items []Item
	conn, err := Paginate(items, getItemID, intPtr(5), nil)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 0 {
		t.Fatalf("got %d edges, want 0", len(conn.Edges))
	}
	if conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be false for empty slice")
	}
	if conn.PageInfo.EndCursor != nil {
		t.Error("expected EndCursor to be nil for empty slice")
	}
}
