package conclusion

import (
	"os"
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	c, err := store.Create("work", "ticket-123", "/repo", "# Summary\n\nDid some work.")
	if err != nil {
		t.Fatal(err)
	}

	if c.ID == "" {
		t.Error("expected non-empty ID")
	}
	if c.Type != TypeWork {
		t.Errorf("expected type work, got %s", c.Type)
	}
	if c.Ticket != "ticket-123" {
		t.Errorf("expected ticket ticket-123, got %s", c.Ticket)
	}
	if c.Repo != "/repo" {
		t.Errorf("expected repo /repo, got %s", c.Repo)
	}

	// Get by ID
	got, err := store.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID {
		t.Errorf("expected ID %s, got %s", c.ID, got.ID)
	}
	if got.Body != c.Body {
		t.Errorf("expected body %q, got %q", c.Body, got.Body)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("work", "t1", "", "First")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Create("research", "t2", "", "Second")
	if err != nil {
		t.Fatal(err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 conclusions, got %d", len(list))
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get("nonexistent")
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("work", "", "", "")
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestListWithOptions_TypeFilter(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("work", "t1", "", "Work conclusion")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Create("research", "t2", "", "Research conclusion")
	if err != nil {
		t.Fatal(err)
	}

	workList, total, err := store.ListWithOptions(ListOptions{Type: "work"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(workList) != 1 {
		t.Fatalf("expected 1 work conclusion, got %d", len(workList))
	}
	if workList[0].Type != TypeWork {
		t.Errorf("expected type work, got %s", workList[0].Type)
	}

	researchList, total, err := store.ListWithOptions(ListOptions{Type: "research"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(researchList) != 1 {
		t.Fatalf("expected 1 research conclusion, got %d", len(researchList))
	}
	if researchList[0].Type != TypeResearch {
		t.Errorf("expected type research, got %s", researchList[0].Type)
	}
}

func TestListWithOptions_Pagination(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	// Create 5 conclusions with staggered times to ensure deterministic order
	for i := range 5 {
		_, err = store.Create("work", "", "", "Conclusion body")
		if err != nil {
			t.Fatal(err)
		}
		// Small sleep to ensure different Created timestamps
		_ = i
		time.Sleep(2 * time.Millisecond)
	}

	// First page: limit=2, offset=0
	page1, total, err := store.ListWithOptions(ListOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(page1))
	}

	// Second page: limit=2, offset=2
	page2, total2, err := store.ListWithOptions(ListOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total2 != 5 {
		t.Errorf("expected total 5, got %d", total2)
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(page2))
	}

	// No overlap between pages
	for _, c1 := range page1 {
		for _, c2 := range page2 {
			if c1.ID == c2.ID {
				t.Errorf("found duplicate ID %s across pages", c1.ID)
			}
		}
	}

	// Last page: limit=2, offset=4
	page3, total3, err := store.ListWithOptions(ListOptions{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatal(err)
	}
	if total3 != 5 {
		t.Errorf("expected total 5, got %d", total3)
	}
	if len(page3) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(page3))
	}
}

func TestNewStoreCreatesDir(t *testing.T) {
	dir := t.TempDir()
	sessDir := dir + "/sessions"

	_, err := NewStore(sessDir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(sessDir)
	if err != nil {
		t.Fatal("sessions dir should exist:", err)
	}
	if !info.IsDir() {
		t.Fatal("sessions path should be a directory")
	}
}
