package roadmap

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var allowedFrontMatterFields = map[string]struct{}{
	"id":         {},
	"title":      {},
	"type":       {},
	"priority":   {},
	"depends_on": {},
}

func parseFrontMatterYAML(yamlContent string) (map[string]yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return nil, fmt.Errorf("invalid YAML front-matter: %w", err)
	}

	fields := map[string]yaml.Node{}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = *root.Content[0]
	}
	if root.Kind == yaml.MappingNode {
		seen := map[string]struct{}{}
		for i := 0; i+1 < len(root.Content); i += 2 {
			keyNode := root.Content[i]
			key := keyNode.Value
			if _, ok := seen[key]; ok {
				return nil, fmt.Errorf("duplicate field %q", key)
			}
			seen[key] = struct{}{}
			fields[key] = *root.Content[i+1]
		}
	}

	return fields, nil
}

func parseTaskFile(path string) (Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Task{}, err
	}

	yamlContent, err := extractFrontMatter(string(data))
	if err != nil {
		return Task{}, err
	}

	fields, err := parseFrontMatterYAML(yamlContent)
	if err != nil {
		return Task{}, err
	}

	if err := validateFrontMatterFields(fields); err != nil {
		return Task{}, err
	}

	id, err := parseScalarField(fields["id"], "id")
	if err != nil {
		return Task{}, err
	}
	if !idPattern.MatchString(id) {
		return Task{}, fmt.Errorf(`id %q does not match required pattern ^[a-z0-9]+(-[a-z0-9]+)*$`, id)
	}

	title, err := parseScalarField(fields["title"], "title")
	if err != nil {
		return Task{}, err
	}

	typ, err := parseScalarField(fields["type"], "type")
	if err != nil {
		return Task{}, err
	}
	if _, ok := validTypes[typ]; !ok {
		return Task{}, fmt.Errorf(`type %q is invalid (expected bug, refactoring, documentation, test, or feature)`, typ)
	}

	priority, err := parseScalarField(fields["priority"], "priority")
	if err != nil {
		return Task{}, err
	}
	if _, ok := validPriorities[priority]; !ok {
		return Task{}, fmt.Errorf(`priority %q is invalid (expected high, medium, or low)`, priority)
	}

	dependsOn, err := parseDependsOn(fields["depends_on"])
	if err != nil {
		return Task{}, err
	}

	fileID := strings.TrimSuffix(filepath.Base(path), ".md")
	if fileID != id {
		return Task{}, fmt.Errorf(`filename %q does not match id %q`, fileID+".md", id)
	}

	return Task{
		Path:      path,
		FileID:    fileID,
		ID:        id,
		Title:     title,
		Type:      typ,
		Priority:  priority,
		DependsOn: dependsOn,
	}, nil
}

func extractFrontMatter(content string) (string, error) {
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

func validateFrontMatterFields(fields map[string]yaml.Node) error {
	var unknown []string
	for key := range fields {
		if _, ok := allowedFrontMatterFields[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown fields %s", quoteFieldNames(unknown))
	}

	for _, key := range []string{"id", "title", "type", "priority", "depends_on"} {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("missing required field %q", key)
		}
	}
	return nil
}

func parseScalarField(node yaml.Node, field string) (string, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return "", fmt.Errorf("%s must not be null", field)
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", fmt.Errorf("%s must be a string", field)
	}
	if node.Value == "" {
		return "", fmt.Errorf("%s must be a non-empty string", field)
	}
	return node.Value, nil
}

func parseDependsOn(node yaml.Node) ([]string, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return nil, fmt.Errorf("depends_on must be an array")
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("depends_on must be an array")
	}

	dependsOn := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			return nil, fmt.Errorf("depends_on must be an array of id strings")
		}
		if item.Value == "" {
			return nil, fmt.Errorf("depends_on must be an array of id strings")
		}
		if !idPattern.MatchString(item.Value) {
			return nil, fmt.Errorf(`depends_on item %q does not match required pattern ^[a-z0-9]+(-[a-z0-9]+)*$`, item.Value)
		}
		dependsOn = append(dependsOn, item.Value)
	}
	return dependsOn, nil
}

func quoteFieldNames(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return strings.Join(quoted, ", ")
}
