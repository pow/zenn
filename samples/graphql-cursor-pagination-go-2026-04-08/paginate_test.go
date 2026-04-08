package pagination

import "testing"

type User struct {
	ID   string
	Name string
}

func userID(u User) string { return u.ID }

var testUsers = []User{
	{ID: "1", Name: "Alice"},
	{ID: "2", Name: "Bob"},
	{ID: "3", Name: "Carol"},
	{ID: "4", Name: "Dave"},
	{ID: "5", Name: "Eve"},
}

func TestPaginateFirstPage(t *testing.T) {
	conn, err := Paginate(testUsers, 2, nil, userID)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 2 {
		t.Errorf("got %d edges, want 2", len(conn.Edges))
	}
	if conn.Edges[0].Node.Name != "Alice" {
		t.Errorf("first node: got %q, want Alice", conn.Edges[0].Node.Name)
	}
	if conn.Edges[1].Node.Name != "Bob" {
		t.Errorf("second node: got %q, want Bob", conn.Edges[1].Node.Name)
	}
	if !conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be true")
	}
}

func TestPaginateWithAfterCursor(t *testing.T) {
	cursor := EncodeCursor("2") // after Bob
	conn, err := Paginate(testUsers, 2, &cursor, userID)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 2 {
		t.Errorf("got %d edges, want 2", len(conn.Edges))
	}
	if conn.Edges[0].Node.Name != "Carol" {
		t.Errorf("first node: got %q, want Carol", conn.Edges[0].Node.Name)
	}
	if conn.Edges[1].Node.Name != "Dave" {
		t.Errorf("second node: got %q, want Dave", conn.Edges[1].Node.Name)
	}
	if !conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be true")
	}
}

func TestPaginateLastPage(t *testing.T) {
	cursor := EncodeCursor("3") // after Carol
	conn, err := Paginate(testUsers, 10, &cursor, userID)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 2 {
		t.Errorf("got %d edges, want 2", len(conn.Edges))
	}
	if conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be false on last page")
	}
}

func TestPaginateEmptyResult(t *testing.T) {
	cursor := EncodeCursor("5") // after Eve (last item)
	conn, err := Paginate(testUsers, 10, &cursor, userID)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if len(conn.Edges) != 0 {
		t.Errorf("got %d edges, want 0", len(conn.Edges))
	}
	if conn.PageInfo.HasNextPage {
		t.Error("expected HasNextPage to be false")
	}
	if conn.PageInfo.EndCursor != nil {
		t.Error("expected EndCursor to be nil for empty result")
	}
}
