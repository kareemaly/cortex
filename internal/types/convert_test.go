package types

import (
	"strings"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/docs"
)

func TestExtractSnippet(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		query  string
		maxLen int
		want   string
	}{
		{
			name:   "empty query",
			body:   "some body text",
			query:  "",
			maxLen: 150,
			want:   "",
		},
		{
			name:   "empty body",
			body:   "",
			query:  "test",
			maxLen: 150,
			want:   "",
		},
		{
			name:   "no match",
			body:   "some body text here",
			query:  "xyz",
			maxLen: 150,
			want:   "",
		},
		{
			name:   "body shorter than maxLen",
			body:   "short body with keyword here",
			query:  "keyword",
			maxLen: 150,
			want:   "short body with keyword here",
		},
		{
			name:   "match at start",
			body:   "keyword" + strings.Repeat("x", 200),
			query:  "keyword",
			maxLen: 50,
			want:   "keyword" + strings.Repeat("x", 43) + "...",
		},
		{
			name:   "match at end",
			body:   strings.Repeat("x", 200) + "keyword",
			query:  "keyword",
			maxLen: 50,
			want:   "..." + strings.Repeat("x", 43) + "keyword",
		},
		{
			name:   "match in middle",
			body:   strings.Repeat("a", 100) + "keyword" + strings.Repeat("b", 100),
			query:  "keyword",
			maxLen: 50,
			want:   "..." + strings.Repeat("a", 21) + "keyword" + strings.Repeat("b", 22) + "...",
		},
		{
			name:   "case insensitive match",
			body:   "some text with KeyWord in it" + strings.Repeat("x", 200),
			query:  "keyword",
			maxLen: 50,
			want:   "some text with KeyWord in it" + strings.Repeat("x", 22) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSnippet(tt.body, tt.query, tt.maxLen)
			if got != tt.want {
				t.Errorf("ExtractSnippet() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToDocSummaryWithQuery(t *testing.T) {
	now := time.Now()
	doc := &docs.Doc{
		ID:       "abc123",
		Title:    "Test Doc",
		Category: "specs",
		Tags:     []string{"api"},
		Created:  now,
		Updated:  now,
		Body:     "This document describes the authentication flow for the API.",
	}

	t.Run("snippet populated when query matches body", func(t *testing.T) {
		s := ToDocSummaryWithQuery(doc, "authentication")
		if s.Snippet == "" {
			t.Error("expected non-empty snippet when query matches body")
		}
		if !strings.Contains(strings.ToLower(s.Snippet), "authentication") {
			t.Errorf("snippet should contain the query term, got %q", s.Snippet)
		}
		if s.ID != "abc123" {
			t.Errorf("ID = %q, want %q", s.ID, "abc123")
		}
	})

	t.Run("snippet empty when no match", func(t *testing.T) {
		s := ToDocSummaryWithQuery(doc, "nonexistent")
		if s.Snippet != "" {
			t.Errorf("expected empty snippet, got %q", s.Snippet)
		}
	})

	t.Run("snippet empty when query is empty", func(t *testing.T) {
		s := ToDocSummaryWithQuery(doc, "")
		if s.Snippet != "" {
			t.Errorf("expected empty snippet, got %q", s.Snippet)
		}
	})
}
