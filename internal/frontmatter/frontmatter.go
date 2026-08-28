// Package frontmatter extracts and parses YAML front-matter from Markdown files.
package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const fence = "---"

// Extract returns the YAML block delimited by the leading `---` fence. Both
// fences must be a line of their own, so a `---` inside the body does not end
// the front-matter.
func Extract(content string) (string, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	lines := strings.Split(content, "\n")
	if trimLine(lines[0]) != fence {
		return "", fmt.Errorf("missing YAML front-matter")
	}

	for i := 1; i < len(lines); i++ {
		if trimLine(lines[i]) == fence {
			return strings.Join(lines[1:i], "\n"), nil
		}
	}
	return "", fmt.Errorf("unclosed YAML front-matter")
}

func trimLine(line string) string {
	return strings.TrimRight(line, " \t\r")
}

// Parse returns the top-level fields of a YAML mapping, keeping the raw nodes so
// that callers can distinguish a missing field from an explicit null. An empty
// block yields no fields, which the caller reports as missing required fields.
func Parse(yamlContent string) (map[string]yaml.Node, error) {
	if strings.TrimSpace(yamlContent) == "" {
		return map[string]yaml.Node{}, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return nil, fmt.Errorf("invalid YAML front-matter: %w", err)
	}

	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = *root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("front-matter must be a YAML mapping")
	}

	fields := map[string]yaml.Node{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if _, ok := fields[key]; ok {
			return nil, fmt.Errorf("duplicate field %q", key)
		}
		fields[key] = Resolve(*root.Content[i+1])
	}
	return fields, nil
}

// Resolve follows an alias node to the anchor it points at, so that a field
// written as `title: *anchor` is checked as the value it stands for.
func Resolve(node yaml.Node) yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = *node.Alias
	}
	return node
}

// ParseFile extracts and parses the front-matter of a Markdown document.
func ParseFile(content string) (map[string]yaml.Node, error) {
	yamlContent, err := Extract(content)
	if err != nil {
		return nil, err
	}
	return Parse(yamlContent)
}
