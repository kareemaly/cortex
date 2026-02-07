package storage

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// ParseFrontmatter parses a markdown file with YAML frontmatter into a typed struct and body.
func ParseFrontmatter[T any](data []byte) (*T, string, error) {
	content := string(data)

	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return nil, "", fmt.Errorf("missing frontmatter delimiter")
	}

	rest := content[len(frontmatterDelimiter)+1:] // skip "---\n"
	endIdx := strings.Index(rest, "\n"+frontmatterDelimiter)
	if endIdx == -1 {
		return nil, "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	yamlContent := rest[:endIdx]
	bodyStart := endIdx + len("\n"+frontmatterDelimiter)
	body := ""
	if bodyStart < len(rest) {
		body = rest[bodyStart:]
		body = strings.TrimPrefix(body, "\n")
	}

	var meta T
	if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
		return nil, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	return &meta, body, nil
}

// SerializeFrontmatter serializes a typed struct and body to markdown with YAML frontmatter.
func SerializeFrontmatter[T any](meta *T, body string) ([]byte, error) {
	yamlData, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterDelimiter + "\n")
	buf.Write(yamlData)
	buf.WriteString(frontmatterDelimiter + "\n")
	if body != "" {
		buf.WriteString(body)
	}

	return buf.Bytes(), nil
}
