package storage

import (
	"testing"
	"time"
)

type sampleMeta struct {
	ID      string    `yaml:"id"`
	Title   string    `yaml:"title"`
	Tags    []string  `yaml:"tags,omitempty"`
	Created time.Time `yaml:"created"`
}

func TestFrontmatterRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	meta := &sampleMeta{
		ID:      "test-123",
		Title:   "Test Title",
		Tags:    []string{"a", "b"},
		Created: now,
	}
	body := "# Heading\n\nSome **markdown** body."

	data, err := SerializeFrontmatter(meta, body)
	if err != nil {
		t.Fatalf("SerializeFrontmatter failed: %v", err)
	}

	parsed, parsedBody, err := ParseFrontmatter[sampleMeta](data)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}

	if parsed.ID != meta.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, meta.ID)
	}
	if parsed.Title != meta.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, meta.Title)
	}
	if len(parsed.Tags) != 2 || parsed.Tags[0] != "a" || parsed.Tags[1] != "b" {
		t.Errorf("Tags = %v, want [a b]", parsed.Tags)
	}
	if !parsed.Created.Equal(meta.Created) {
		t.Errorf("Created = %v, want %v", parsed.Created, meta.Created)
	}
	if parsedBody != body {
		t.Errorf("body = %q, want %q", parsedBody, body)
	}
}

func TestFrontmatterEmptyBody(t *testing.T) {
	meta := &sampleMeta{ID: "test", Title: "Empty Body"}
	data, err := SerializeFrontmatter(meta, "")
	if err != nil {
		t.Fatalf("SerializeFrontmatter failed: %v", err)
	}

	parsed, body, err := ParseFrontmatter[sampleMeta](data)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}

	if parsed.ID != "test" {
		t.Errorf("ID = %q, want %q", parsed.ID, "test")
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestFrontmatterMissingOpeningDelimiter(t *testing.T) {
	_, _, err := ParseFrontmatter[sampleMeta]([]byte("no frontmatter here"))
	if err == nil {
		t.Error("expected error for missing opening delimiter")
	}
}

func TestFrontmatterMissingClosingDelimiter(t *testing.T) {
	_, _, err := ParseFrontmatter[sampleMeta]([]byte("---\nid: test\n"))
	if err == nil {
		t.Error("expected error for missing closing delimiter")
	}
}

func TestFrontmatterInvalidYAML(t *testing.T) {
	data := []byte("---\n: invalid: yaml: [[\n---\nbody")
	_, _, err := ParseFrontmatter[sampleMeta](data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestFrontmatterNoTags(t *testing.T) {
	meta := &sampleMeta{ID: "test", Title: "No Tags"}
	data, err := SerializeFrontmatter(meta, "body")
	if err != nil {
		t.Fatalf("SerializeFrontmatter failed: %v", err)
	}

	parsed, _, err := ParseFrontmatter[sampleMeta](data)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}

	if parsed.Tags != nil {
		t.Errorf("Tags = %v, want nil", parsed.Tags)
	}
}
