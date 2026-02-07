package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kareemaly/cortex/internal/storage"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "docs-store-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	store, err := NewStore(tmpDir, nil, "")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create store: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestNewStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-store-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	docsDir := filepath.Join(tmpDir, "docs")
	store, err := NewStore(docsDir, nil, "")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	info, err := os.Stat(store.docsDir)
	if err != nil {
		t.Fatalf("docs directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("docs path is not a directory")
	}
}

func TestStoreCreate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, err := store.Create("API Design", "specs", "# API\nOverview", []string{"api"}, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if doc.ID == "" {
		t.Error("doc ID should not be empty")
	}
	if doc.Title != "API Design" {
		t.Errorf("title = %q, want %q", doc.Title, "API Design")
	}
	if doc.Category != "specs" {
		t.Errorf("category = %q, want %q", doc.Category, "specs")
	}
	if doc.Body != "# API\nOverview" {
		t.Errorf("body = %q, want %q", doc.Body, "# API\nOverview")
	}
	if len(doc.Tags) != 1 || doc.Tags[0] != "api" {
		t.Errorf("tags = %v, want [api]", doc.Tags)
	}
	if doc.Created.IsZero() {
		t.Error("created date should be set")
	}
	if doc.Updated.IsZero() {
		t.Error("updated date should be set")
	}
}

func TestStoreCreateCategoryDir(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Create("Test Doc", "decisions", "body", nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	catDir := filepath.Join(store.docsDir, "decisions")
	info, err := os.Stat(catDir)
	if err != nil {
		t.Fatalf("category directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("category path is not a directory")
	}
}

func TestStoreCreateEmptyTitle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Create("", "specs", "body", nil, nil)
	if err == nil {
		t.Error("expected error for empty title")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreCreateEmptyCategory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Create("Title", "", "body", nil, nil)
	if err == nil {
		t.Error("expected error for empty category")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	created, err := store.Create("Test Doc", "specs", "body content", []string{"tag1"}, []string{"ticket:abc"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, created.ID)
	}
	if retrieved.Title != "Test Doc" {
		t.Errorf("title = %q, want %q", retrieved.Title, "Test Doc")
	}
	if retrieved.Category != "specs" {
		t.Errorf("category = %q, want %q", retrieved.Category, "specs")
	}
	if retrieved.Body != "body content" {
		t.Errorf("body = %q, want %q", retrieved.Body, "body content")
	}
	if len(retrieved.Tags) != 1 || retrieved.Tags[0] != "tag1" {
		t.Errorf("tags = %v, want [tag1]", retrieved.Tags)
	}
	if len(retrieved.References) != 1 || retrieved.References[0] != "ticket:abc" {
		t.Errorf("references = %v, want [ticket:abc]", retrieved.References)
	}
}

func TestStoreGetLoadsComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)
	_, _ = store.AddComment(doc.ID, "claude", storage.CommentGeneral, "comment 1", nil)
	_, _ = store.AddComment(doc.ID, "human", storage.CommentGeneral, "comment 2", nil)

	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(retrieved.Comments) != 2 {
		t.Errorf("comments count = %d, want 2", len(retrieved.Comments))
	}
}

func TestStoreListDoesNotLoadComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)
	_, _ = store.AddComment(doc.ID, "claude", storage.CommentGeneral, "comment", nil)

	docs, err := store.List("", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	if len(docs[0].Comments) != 0 {
		t.Errorf("List should not load comments, got %d", len(docs[0].Comments))
	}
}

func TestStoreGetNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Get("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent doc")
	}
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Original Title", "specs", "Original body", nil, nil)

	newTitle := "Updated Title"
	newBody := "Updated body"
	updated, err := store.Update(doc.ID, &newTitle, &newBody, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Body != newBody {
		t.Errorf("body = %q, want %q", updated.Body, newBody)
	}
	if updated.Updated.Before(doc.Updated) || updated.Updated.Equal(doc.Updated) {
		t.Error("updated timestamp should advance")
	}
}

func TestStoreUpdatePartial(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Original Title", "specs", "Original body", []string{"tag1"}, nil)

	newTitle := "Updated Title"
	updated, err := store.Update(doc.ID, &newTitle, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Body != "Original body" {
		t.Error("body should remain unchanged")
	}
}

func TestStoreUpdateTags(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Doc", "specs", "body", []string{"old"}, nil)

	newTags := []string{"new1", "new2"}
	updated, err := store.Update(doc.ID, nil, nil, &newTags, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if len(updated.Tags) != 2 || updated.Tags[0] != "new1" || updated.Tags[1] != "new2" {
		t.Errorf("tags = %v, want [new1 new2]", updated.Tags)
	}
}

func TestStoreUpdateReferences(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Doc", "specs", "body", nil, nil)

	newRefs := []string{"ticket:123", "doc:456"}
	updated, err := store.Update(doc.ID, nil, nil, nil, &newRefs)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if len(updated.References) != 2 || updated.References[0] != "ticket:123" {
		t.Errorf("references = %v, want [ticket:123 doc:456]", updated.References)
	}
}

func TestStoreUpdateEmptyTitle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Doc", "specs", "body", nil, nil)

	emptyTitle := ""
	_, err := store.Update(doc.ID, &emptyTitle, nil, nil, nil)
	if err == nil {
		t.Error("expected error for empty title")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreUpdateTitleRename(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Old Title", "specs", "body", nil, nil)
	shortID := storage.ShortID(doc.ID)

	// Verify old dir exists
	oldDir := filepath.Join(store.docsDir, "specs", "old-title-"+shortID)
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old dir should exist: %v", err)
	}

	newTitle := "New Title"
	_, err := store.Update(doc.ID, &newTitle, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Old dir should be gone
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old dir should be removed after title change")
	}

	// New dir should exist
	newDir := filepath.Join(store.docsDir, "specs", "new-title-"+shortID)
	if _, err := os.Stat(newDir); err != nil {
		t.Errorf("new dir should exist: %v", err)
	}

	// Should still be retrievable
	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get after rename failed: %v", err)
	}
	if retrieved.Title != "New Title" {
		t.Errorf("title = %q, want %q", retrieved.Title, "New Title")
	}
}

func TestStoreDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)

	if err := store.Delete(doc.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(doc.ID)
	if !IsNotFound(err) {
		t.Error("doc should not exist after delete")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Delete("nonexistent")
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestStoreMove(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)

	moved, err := store.Move(doc.ID, "decisions")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if moved.Category != "decisions" {
		t.Errorf("category = %q, want %q", moved.Category, "decisions")
	}

	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Category != "decisions" {
		t.Errorf("category after get = %q, want %q", retrieved.Category, "decisions")
	}
}

func TestStoreMoveSameCategory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)

	moved, err := store.Move(doc.ID, "specs")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if moved.Category != "specs" {
		t.Errorf("category = %q, want %q", moved.Category, "specs")
	}
}

func TestStoreMoveEmptyCategory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)

	_, err := store.Move(doc.ID, "")
	if err == nil {
		t.Error("expected error for empty category")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("Doc 1", "specs", "body 1", nil, nil)
	_, _ = store.Create("Doc 2", "specs", "body 2", nil, nil)
	_, _ = store.Create("Doc 3", "decisions", "body 3", nil, nil)

	docs, err := store.List("", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("len(docs) = %d, want 3", len(docs))
	}
}

func TestStoreListByCategory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("Spec Doc", "specs", "body", nil, nil)
	_, _ = store.Create("Decision Doc", "decisions", "body", nil, nil)

	docs, err := store.List("specs", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1", len(docs))
	}
	if docs[0].Title != "Spec Doc" {
		t.Errorf("title = %q, want %q", docs[0].Title, "Spec Doc")
	}
}

func TestStoreListByTag(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("Doc A", "specs", "body", []string{"api", "v2"}, nil)
	_, _ = store.Create("Doc B", "specs", "body", []string{"internal"}, nil)

	docs, err := store.List("", "api", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1", len(docs))
	}
	if docs[0].Title != "Doc A" {
		t.Errorf("title = %q, want %q", docs[0].Title, "Doc A")
	}
}

func TestStoreListByQuery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("API Design", "specs", "REST endpoints", nil, nil)
	_, _ = store.Create("DB Schema", "specs", "Tables and indexes", nil, nil)

	// Match by title
	docs, err := store.List("", "", "api")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1", len(docs))
	}

	// Match by body
	docs, err = store.List("", "", "indexes")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1", len(docs))
	}
}

func TestStoreListEmpty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	docs, err := store.List("", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if docs == nil {
		t.Error("docs should not be nil")
	}
	if len(docs) != 0 {
		t.Errorf("len(docs) = %d, want 0", len(docs))
	}
}

func TestStoreListNonexistentCategory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	docs, err := store.List("nonexistent", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("len(docs) = %d, want 0", len(docs))
	}
}

func TestFrontmatterRoundTrip(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, err := store.Create("Test Title", "specs", "# Heading\n\nBody with **markdown**.", []string{"tag1", "tag2"}, []string{"ticket:abc"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != doc.Title {
		t.Errorf("title = %q, want %q", retrieved.Title, doc.Title)
	}
	if retrieved.Body != doc.Body {
		t.Errorf("body = %q, want %q", retrieved.Body, doc.Body)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("tags length = %d, want 2", len(retrieved.Tags))
	}
	if len(retrieved.References) != 1 || retrieved.References[0] != "ticket:abc" {
		t.Errorf("references = %v, want [ticket:abc]", retrieved.References)
	}
}

func TestCategoryNotInFrontmatter(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)
	shortID := storage.ShortID(doc.ID)

	// Read raw index.md file
	indexPath := filepath.Join(store.docsDir, "specs", storage.DirName(doc.Title, doc.ID, "doc"), "index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.md: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "category:") {
		t.Error("category should NOT be in frontmatter")
	}
	if !strings.Contains(content, "id: "+doc.ID) {
		t.Errorf("expected id in frontmatter, shortID=%s", shortID)
	}

	// But Category should be populated on load
	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Category != "specs" {
		t.Errorf("category = %q, want %q (derived from path)", retrieved.Category, "specs")
	}
}

func TestStoreDirLayout(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("API Design", "specs", "body", nil, nil)
	shortID := storage.ShortID(doc.ID)

	entityDir := filepath.Join(store.docsDir, "specs", "api-design-"+shortID)
	if _, err := os.Stat(entityDir); err != nil {
		t.Fatalf("entity dir should exist: %v", err)
	}

	indexPath := filepath.Join(entityDir, "index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.md should exist: %v", err)
	}
}

func TestStoreCreateWithReferences(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	refs := []string{"ticket:abc123", "doc:xyz789"}
	doc, err := store.Create("Linked Doc", "specs", "body", nil, refs)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(doc.References) != 2 {
		t.Errorf("references length = %d, want 2", len(doc.References))
	}

	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(retrieved.References) != 2 || retrieved.References[0] != "ticket:abc123" {
		t.Errorf("references = %v, want [ticket:abc123 doc:xyz789]", retrieved.References)
	}
}

func TestStoreListTagCaseInsensitive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("Doc", "specs", "body", []string{"API"}, nil)

	docs, err := store.List("", "api", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1 (tag filter should be case-insensitive)", len(docs))
	}
}

func TestStoreListQueryCaseInsensitive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("API Design", "specs", "body", nil, nil)

	docs, err := store.List("", "", "api design")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len(docs) = %d, want 1 (query should be case-insensitive)", len(docs))
	}
}

func TestStoreAddComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)

	comment, err := store.AddComment(doc.ID, "claude", storage.CommentGeneral, "Test comment", nil)
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if comment.Author != "claude" {
		t.Errorf("author = %q, want %q", comment.Author, "claude")
	}
	if comment.Content != "Test comment" {
		t.Errorf("content = %q, want %q", comment.Content, "Test comment")
	}
}

func TestStoreListComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	doc, _ := store.Create("Test Doc", "specs", "body", nil, nil)
	_, _ = store.AddComment(doc.ID, "claude", storage.CommentGeneral, "First", nil)
	_, _ = store.AddComment(doc.ID, "human", storage.CommentGeneral, "Second", nil)

	comments, err := store.ListComments(doc.ID)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, want 2", len(comments))
	}
	if comments[0].Content != "First" {
		t.Errorf("first comment = %q, want %q", comments[0].Content, "First")
	}
}

func TestContainsTag(t *testing.T) {
	tests := []struct {
		tags []string
		tag  string
		want bool
	}{
		{[]string{"api", "v2"}, "api", true},
		{[]string{"api", "v2"}, "API", true},
		{[]string{"api", "v2"}, "v3", false},
		{nil, "api", false},
		{[]string{}, "api", false},
	}

	for _, tt := range tests {
		got := containsTag(tt.tags, tt.tag)
		if got != tt.want {
			t.Errorf("containsTag(%v, %q) = %v, want %v", tt.tags, tt.tag, got, tt.want)
		}
	}
}
