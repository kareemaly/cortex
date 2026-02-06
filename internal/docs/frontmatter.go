package docs

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// ParseDoc parses a markdown file with YAML frontmatter into a Doc.
// Format:
//
//	---
//	id: ...
//	title: ...
//	---
//	Body content here
func ParseDoc(data []byte) (*Doc, error) {
	content := string(data)

	// Must start with ---
	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return nil, fmt.Errorf("missing frontmatter delimiter")
	}

	// Find the closing ---
	rest := content[len(frontmatterDelimiter)+1:] // skip "---\n"
	endIdx := strings.Index(rest, "\n"+frontmatterDelimiter)
	if endIdx == -1 {
		return nil, fmt.Errorf("missing closing frontmatter delimiter")
	}

	yamlContent := rest[:endIdx]
	// Body starts after the closing "---\n"
	bodyStart := endIdx + len("\n"+frontmatterDelimiter)
	body := ""
	if bodyStart < len(rest) {
		body = rest[bodyStart:]
		// Strip leading newline after closing ---
		body = strings.TrimPrefix(body, "\n")
	}

	var doc Doc
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	doc.Body = body
	return &doc, nil
}

// SerializeDoc serializes a Doc to markdown with YAML frontmatter.
func SerializeDoc(doc *Doc) ([]byte, error) {
	// Build frontmatter struct (excludes Body via yaml:"-")
	yamlData, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterDelimiter + "\n")
	buf.Write(yamlData)
	buf.WriteString(frontmatterDelimiter + "\n")
	if doc.Body != "" {
		buf.WriteString(doc.Body)
	}

	return buf.Bytes(), nil
}
