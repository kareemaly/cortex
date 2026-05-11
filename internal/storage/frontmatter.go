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
	yamlContent, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, "", err
	}

	var meta T
	if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
		return nil, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	return &meta, body, nil
}

// FrontmatterKeys returns the top-level frontmatter keys in a markdown file.
func FrontmatterKeys(data []byte) (map[string]struct{}, error) {
	yamlContent, _, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	if len(doc.Content) == 0 {
		return map[string]struct{}{}, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse frontmatter: expected top-level mapping")
	}

	keys := make(map[string]struct{}, len(root.Content)/2)
	for i := 0; i < len(root.Content); i += 2 {
		keys[root.Content[i].Value] = struct{}{}
	}
	return keys, nil
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

func splitFrontmatter(data []byte) (string, string, error) {
	content := string(data)

	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return "", "", fmt.Errorf("missing frontmatter delimiter")
	}

	rest := content[len(frontmatterDelimiter)+1:] // skip "---\n"
	endIdx := strings.Index(rest, "\n"+frontmatterDelimiter)
	if endIdx == -1 {
		return "", "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	yamlContent := rest[:endIdx]
	bodyStart := endIdx + len("\n"+frontmatterDelimiter)
	body := ""
	if bodyStart < len(rest) {
		body = rest[bodyStart:]
		body = strings.TrimPrefix(body, "\n")
	}

	return yamlContent, body, nil
}
