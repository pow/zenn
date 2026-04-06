package pagination_test

import (
	"testing"

	"github.com/example/relay-cursor-pagination/cursor"
	"github.com/example/relay-cursor-pagination/pagination"
)

// User はテスト用の Node 実装
type User struct {
	ID    string
	Name  string
	Email string
}

func (u User) GetID() string { return u.ID }

func makeUsers(ids ...string) []User {
	users := make([]User, len(ids))
	for i, id := range ids {
		users[i] = User{ID: id, Name: "user-" + id, Email: id + "@example.com"}
	}
	return users
}

func TestBuild_HasNextPage(t *testing.T) {
	// first=2 に対して 3件（first+1）渡す → hasNextPage=true
	users := makeUsers("1", "2", "3")
	conn := pagination.Build(users, 2, 10)

	if !conn.PageInfo.HasNextPage {
		t.Error("HasNextPage should be true when nodes > first")
	}
	if len(conn.Edges) != 2 {
		t.Errorf("Edges count got %d, want 2", len(conn.Edges))
	}
	// 余分な1件（ID=3）が切り落とされていること
	if conn.Edges[1].Node.ID != "2" {
		t.Errorf("Last edge ID got %q, want %q", conn.Edges[1].Node.ID, "2")
	}
}

func TestBuild_NoNextPage(t *testing.T) {
	// first=3 に対して 2件渡す → hasNextPage=false
	users := makeUsers("1", "2")
	conn := pagination.Build(users, 3, 2)

	if conn.PageInfo.HasNextPage {
		t.Error("HasNextPage should be false when nodes <= first")
	}
	if len(conn.Edges) != 2 {
		t.Errorf("Edges count got %d, want 2", len(conn.Edges))
	}
}

func TestBuild_ExactlyFirst(t *testing.T) {
	// first=3 に対してちょうど 3件渡す → hasNextPage=false
	users := makeUsers("1", "2", "3")
	conn := pagination.Build(users, 3, 3)

	if conn.PageInfo.HasNextPage {
		t.Error("HasNextPage should be false when nodes == first")
	}
	if len(conn.Edges) != 3 {
		t.Errorf("Edges count got %d, want 3", len(conn.Edges))
	}
}

func TestBuild_EmptyResult(t *testing.T) {
	conn := pagination.Build([]User{}, 10, 0)

	if conn.PageInfo.HasNextPage {
		t.Error("HasNextPage should be false for empty result")
	}
	if conn.PageInfo.EndCursor != nil {
		t.Error("EndCursor should be nil for empty result")
	}
	if len(conn.Edges) != 0 {
		t.Errorf("Edges count got %d, want 0", len(conn.Edges))
	}
	if conn.TotalCount != 0 {
		t.Errorf("TotalCount got %d, want 0", conn.TotalCount)
	}
}

func TestBuild_EndCursorIsDecodable(t *testing.T) {
	users := makeUsers("abc-123", "def-456")
	conn := pagination.Build(users, 5, 2)

	if conn.PageInfo.EndCursor == nil {
		t.Fatal("EndCursor should not be nil")
	}

	// EndCursor は最後のノードの ID をエンコードしたもの
	decoded, err := cursor.Decode(*conn.PageInfo.EndCursor)
	if err != nil {
		t.Fatalf("EndCursor should be decodable: %v", err)
	}
	if decoded != "def-456" {
		t.Errorf("EndCursor decoded to %q, want %q", decoded, "def-456")
	}
}

func TestBuild_TotalCount(t *testing.T) {
	users := makeUsers("1", "2", "3")
	conn := pagination.Build(users, 2, 100)

	if conn.TotalCount != 100 {
		t.Errorf("TotalCount got %d, want 100", conn.TotalCount)
	}
}

func TestBuild_CursorPagination(t *testing.T) {
	// ページ1: first=2, 3件取得 → 2件返す + hasNext=true
	page1Users := makeUsers("1", "2", "3")
	page1 := pagination.Build(page1Users, 2, 5)

	if !page1.PageInfo.HasNextPage {
		t.Error("Page 1 should have next page")
	}

	// EndCursor をデコードして次のページの after として使う
	afterID, err := cursor.Decode(*page1.PageInfo.EndCursor)
	if err != nil {
		t.Fatalf("Failed to decode page 1 EndCursor: %v", err)
	}
	if afterID != "2" {
		t.Errorf("After cursor ID got %q, want %q", afterID, "2")
	}

	// ページ2: afterID="2" の次から first=2 件 → "3", "4", "5" のうち2件 + hasNext=true
	page2Users := makeUsers("3", "4", "5")
	page2 := pagination.Build(page2Users, 2, 5)

	if !page2.PageInfo.HasNextPage {
		t.Error("Page 2 should have next page")
	}

	// ページ3: afterID="4" の次から first=2 件 → "5" のみ → hasNext=false
	page3Users := makeUsers("5")
	page3 := pagination.Build(page3Users, 2, 5)

	if page3.PageInfo.HasNextPage {
		t.Error("Page 3 should not have next page")
	}
	if len(page3.Edges) != 1 {
		t.Errorf("Page 3 edges count got %d, want 1", len(page3.Edges))
	}
}
