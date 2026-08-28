// Package frontmatter extracts and parses YAML front-matter from Markdown files.
package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Extract returns the YAML block delimited by the leading `---` fence.
func Extract(content string) (string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("missing YAML front-matter")
	}

	rest := content[len("---"):]
	switch {
	case strings.HasPrefix(rest, "\r\n"):
		rest = rest[2:]
	case strings.HasPrefix(rest, "\n"):
		rest = rest[1:]
	default:
		return "", fmt.Errorf("missing YAML front-matter")
	}

	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", fmt.Errorf("unclosed YAML front-matter")
	}
	return rest[:end], nil
}

// Parse returns the top-level fields of a YAML mapping, keeping the raw nodes so
// that callers can distinguish a missing field from an explicit null. A document
// that is not a mapping has no fields.
func Parse(yamlContent string) (map[string]yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return nil, fmt.Errorf("invalid YAML front-matter: %w", err)
	}

	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = *root.Content[0]
	}
	fields := map[string]yaml.Node{}
	if root.Kind != yaml.MappingNode {
		return fields, nil
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if _, ok := fields[key]; ok {
			return nil, fmt.Errorf("duplicate field %q", key)
		}
		fields[key] = *root.Content[i+1]
	}
	return fields, nil
}

// ParseFile extracts and parses the front-matter of a Markdown document.
func ParseFile(content string) (map[string]yaml.Node, error) {
	yamlContent, err := Extract(content)
	if err != nil {
		return nil, err
	}
	return Parse(yamlContent)
}
