package storage

import (
	"os"
	"testing"
	"time"
)

func TestCreateComment(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	comment, err := CreateComment(dir, "claude", CommentGeneral, "Test comment", nil)
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	if comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if comment.Author != "claude" {
		t.Errorf("author = %q, want %q", comment.Author, "claude")
	}
	if comment.Type != CommentGeneral {
		t.Errorf("type = %q, want %q", comment.Type, CommentGeneral)
	}
	if comment.Content != "Test comment" {
		t.Errorf("content = %q, want %q", comment.Content, "Test comment")
	}
	if comment.Created.IsZero() {
		t.Error("created time should be set")
	}
}

func TestCreateCommentEmptyContent(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	_, err = CreateComment(dir, "claude", CommentGeneral, "", nil)
	if err == nil {
		t.Error("expected error for empty content")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateCommentWithAction(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	action := &CommentAction{
		Type: "git_diff",
		Args: GitDiffArgs{RepoPath: "/path/to/repo", Commit: "abc123"},
	}
	comment, err := CreateComment(dir, "claude", CommentReviewRequested, "Review changes", action)
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	if comment.Action == nil {
		t.Fatal("action should not be nil")
	}
	if comment.Action.Type != "git_diff" {
		t.Errorf("action type = %q, want %q", comment.Action.Type, "git_diff")
	}
}

func TestListComments(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	_, _ = CreateComment(dir, "claude", CommentGeneral, "First comment", nil)
	time.Sleep(time.Millisecond) // ensure different timestamps
	_, _ = CreateComment(dir, "human", CommentBlocker, "Second comment", nil)

	comments, err := ListComments(dir)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, want 2", len(comments))
	}

	// Verify sort order: first created should come first
	if comments[0].Content != "First comment" {
		t.Errorf("first comment content = %q, want %q", comments[0].Content, "First comment")
	}
	if comments[1].Content != "Second comment" {
		t.Errorf("second comment content = %q, want %q", comments[1].Content, "Second comment")
	}
}

func TestListCommentsEmpty(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	comments, err := ListComments(dir)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	if comments == nil {
		t.Error("comments should not be nil")
	}
	if len(comments) != 0 {
		t.Errorf("len(comments) = %d, want 0", len(comments))
	}
}

func TestListCommentsMissingDir(t *testing.T) {
	comments, err := ListComments("/nonexistent/path")
	if err != nil {
		t.Fatalf("ListComments should handle missing dir: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("len(comments) = %d, want 0", len(comments))
	}
}

func TestCommentActionRoundTrip(t *testing.T) {
	dir, err := os.MkdirTemp("", "comment-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	action := &CommentAction{
		Type: "git_diff",
		Args: GitDiffArgs{RepoPath: "/repo", Commit: "abc"},
	}
	_, err = CreateComment(dir, "claude", CommentReviewRequested, "Review", action)
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	comments, err := ListComments(dir)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("len(comments) = %d, want 1", len(comments))
	}

	c := comments[0]
	if c.Action == nil {
		t.Fatal("action should not be nil after round-trip")
	}
	if c.Action.Type != "git_diff" {
		t.Errorf("action type = %q, want %q", c.Action.Type, "git_diff")
	}
}
