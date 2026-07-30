package dataserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPutAndGet(t *testing.T) {
	s := NewStore(5 * time.Minute)
	columns := []Field{{Name: "name", Type: "string"}, {Name: "value", Type: "number"}}
	rows := [][]string{{"foo", "1"}, {"bar", "2"}}

	res, err := s.Put("test", "application/json", columns, rows)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if res.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", res.RowCount)
	}
	if len(res.Rows) != 2 {
		t.Errorf("len(Rows) = %d, want 2", len(res.Rows))
	}

	got, ok := s.Get(res.ID)
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Label != "test" {
		t.Errorf("Label = %q, want %q", got.Label, "test")
	}
}

func TestGetNotFound(t *testing.T) {
	s := NewStore(5 * time.Minute)
	_, ok := s.Get(uuid.New())
	if ok {
		t.Error("Get for nonexistent ID returned true")
	}
}

func TestPage(t *testing.T) {
	s := NewStore(5 * time.Minute)
	rows := make([][]string, 250)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("row-%d", i+1)}
	}
	res, _ := s.Put("paging-test", "text/plain", []Field{{Name: "data"}}, rows)

	page, cursor, hasMore := s.Page(res.ID, 0, 100)
	if len(page) != 100 {
		t.Errorf("first page len = %d, want 100", len(page))
	}
	if !hasMore {
		t.Error("hasMore should be true for page 1 of 250")
	}
	if page[0].ID != 1 {
		t.Errorf("first row ID = %d, want 1", page[0].ID)
	}

	page2, cursor2, hasMore2 := s.Page(res.ID, cursor, 100)
	if len(page2) != 100 {
		t.Errorf("second page len = %d, want 100", len(page2))
	}
	if !hasMore2 {
		t.Error("hasMore should be true for page 2 of 250")
	}

	page3, _, hasMore3 := s.Page(res.ID, cursor2, 100)
	if len(page3) != 50 {
		t.Errorf("third page len = %d, want 50", len(page3))
	}
	if hasMore3 {
		t.Error("hasMore should be false for last page")
	}
}

func TestPageEmptyStore(t *testing.T) {
	s := NewStore(5 * time.Minute)
	_, _, hasMore := s.Page(uuid.New(), 0, 100)
	if hasMore {
		t.Error("hasMore should be false for nonexistent resource")
	}
}

func TestPageCursorPastEnd(t *testing.T) {
	s := NewStore(5 * time.Minute)
	res, _ := s.Put("test", "", []Field{{Name: "x"}}, [][]string{{"a"}, {"b"}, {"c"}})
	page, cursor, hasMore := s.Page(res.ID, 999, 100)
	if len(page) != 0 {
		t.Errorf("page len = %d, want 0", len(page))
	}
	if hasMore {
		t.Error("hasMore should be false")
	}
	if cursor != 0 {
		t.Errorf("cursor = %d, want 0", cursor)
	}
}

func TestList(t *testing.T) {
	s := NewStore(5 * time.Minute)
	res1, _ := s.Put("a", "", nil, nil)
	res2, _ := s.Put("b", "", nil, nil)
	list := s.List()
	if len(list) != 2 {
		t.Errorf("List len = %d, want 2", len(list))
	}
	if list[0].ID != res1.ID {
		t.Errorf("first item ID mismatch")
	}
	_ = res2
}

func TestPinAndDelete(t *testing.T) {
	s := NewStore(5 * time.Minute)
	res, _ := s.Put("test", "", nil, nil)

	if err := s.Pin(res.ID, nil); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	r, _ := s.Get(res.ID)
	if !r.Pinned {
		t.Error("expected pinned=true")
	}
	if r.MaxAge != nil {
		t.Errorf("MaxAge = %v, want nil", r.MaxAge)
	}

	maxAge := 15 * time.Minute
	if err := s.Pin(res.ID, &maxAge); err != nil {
		t.Fatalf("Pin with maxAge: %v", err)
	}
	r, _ = s.Get(res.ID)
	if !r.Pinned {
		t.Error("expected pinned=true")
	}
	if *r.MaxAge != maxAge {
		t.Errorf("MaxAge = %v, want %v", *r.MaxAge, maxAge)
	}

	if err := s.Delete(res.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok := s.Get(res.ID)
	if ok {
		t.Error("resource still exists after Delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := NewStore(5 * time.Minute)
	if err := s.Delete(uuid.New()); err == nil {
		t.Error("expected error deleting nonexistent")
	}
}

func TestSweep(t *testing.T) {
	frozen := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	s := NewStore(1 * time.Minute)
	s.clock = func() time.Time { return frozen }

	unpinned, _ := s.Put("expired-unpinned", "", nil, nil)

	pinned, _ := s.Put("pinned-no-maxage", "", nil, nil)
	s.Pin(pinned.ID, nil)

	maxAge := 30 * time.Minute
	pinnedMax, _ := s.Put("pinned-with-maxage", "", nil, nil)
	s.Pin(pinnedMax.ID, &maxAge)

	n := s.Sweep(frozen.Add(2 * time.Minute))
	if n != 1 {
		t.Errorf("Sweep removed %d, want 1 (only expired unpinned)", n)
	}
	if _, ok := s.Get(unpinned.ID); ok {
		t.Error("expired unpinned not swept")
	}
	if _, ok := s.Get(pinned.ID); !ok {
		t.Error("pinned (no maxage) should survive")
	}
	if _, ok := s.Get(pinnedMax.ID); !ok {
		t.Error("pinned (within maxage) should survive")
	}

	n = s.Sweep(frozen.Add(45 * time.Minute))
	if n != 1 {
		t.Errorf("Sweep removed %d, want 1 (pinned beyond maxage)", n)
	}
	if _, ok := s.Get(pinnedMax.ID); ok {
		t.Error("pinned beyond maxage should be swept")
	}
}
