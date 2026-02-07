package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CommentType represents the type of comment.
type CommentType string

const (
	CommentReviewRequested CommentType = "review_requested"
	CommentDone            CommentType = "done"
	CommentBlocker         CommentType = "blocker"
	CommentGeneral         CommentType = "comment"
)

// CommentMeta holds the YAML frontmatter fields for a comment.
type CommentMeta struct {
	ID      string         `yaml:"id"`
	Author  string         `yaml:"author"`
	Type    CommentType    `yaml:"type"`
	Created time.Time      `yaml:"created"`
	Action  *CommentAction `yaml:"action,omitempty"`
}

// Comment represents a comment with frontmatter metadata and markdown body.
type Comment struct {
	CommentMeta
	Content string
}

// CommentAction holds structured data for actionable comments.
type CommentAction struct {
	Type string `yaml:"type"`
	Args any    `yaml:"args"`
}

// GitDiffArgs holds the arguments for a git_diff action.
type GitDiffArgs struct {
	RepoPath string `yaml:"repo_path"`
	Commit   string `yaml:"commit,omitempty"`
}

// CreateComment creates a comment file in entityDir.
func CreateComment(entityDir, author string, commentType CommentType, content string, action *CommentAction) (*Comment, error) {
	if content == "" {
		return nil, &ValidationError{Field: "content", Message: "cannot be empty"}
	}

	now := time.Now().UTC()
	id := uuid.New().String()
	comment := &Comment{
		CommentMeta: CommentMeta{
			ID:      id,
			Author:  author,
			Type:    commentType,
			Created: now,
			Action:  action,
		},
		Content: content,
	}

	data, err := SerializeFrontmatter(&comment.CommentMeta, content)
	if err != nil {
		return nil, fmt.Errorf("serialize comment: %w", err)
	}

	filename := fmt.Sprintf("comment-%s.md", ShortID(id))
	target := filepath.Join(entityDir, filename)

	if err := AtomicWriteFile(target, data); err != nil {
		return nil, fmt.Errorf("write comment: %w", err)
	}

	return comment, nil
}

// ListComments scans entityDir for comment-*.md files, parses them, and returns sorted by created time.
func ListComments(entityDir string) ([]Comment, error) {
	entries, err := os.ReadDir(entityDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Comment{}, nil
		}
		return nil, fmt.Errorf("read directory: %w", err)
	}

	comments := []Comment{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "comment-") || !strings.HasSuffix(name, ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(entityDir, name))
		if err != nil {
			return nil, fmt.Errorf("read comment %s: %w", name, err)
		}

		meta, body, err := ParseFrontmatter[CommentMeta](data)
		if err != nil {
			return nil, fmt.Errorf("parse comment %s: %w", name, err)
		}

		comments = append(comments, Comment{
			CommentMeta: *meta,
			Content:     body,
		})
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].Created.Before(comments[j].Created)
	})

	return comments, nil
}
