// Package lint applies the configured rules to Markdown files.
package lint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	"github.com/syou6162/docs-lint/internal/config"
	"github.com/syou6162/docs-lint/internal/frontmatter"
)

// Issue is a single problem found in a Markdown file.
type Issue struct {
	File    string
	Message string
}

func (i Issue) String() string {
	if i.File == "" {
		return i.Message
	}
	return fmt.Sprintf("%s: %s", i.File, i.Message)
}

// document holds the front-matter values of one file, already type-checked.
type document struct {
	path   string
	values map[string]string
	lists  map[string][]string
}

// Run lints every Markdown file under root against cfg. Paths in the returned
// issues are relative to root, matching the paths the include patterns see.
func Run(root string, cfg *config.Config) ([]Issue, error) {
	files, err := collectMarkdown(root)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	for _, rule := range cfg.Rules {
		matched, err := matchRule(rule, files)
		if err != nil {
			return nil, err
		}
		issues = append(issues, lintRule(root, rule, matched)...)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		return issues[i].Message < issues[j].Message
	})
	return issues, nil
}

func collectMarkdown(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func matchRule(rule config.Rule, files []string) ([]string, error) {
	var matched []string
	for _, file := range files {
		included, err := matchAny(rule.Include, file)
		if err != nil {
			return nil, fmt.Errorf("rule %q: include: %w", rule.Name, err)
		}
		if !included {
			continue
		}
		excluded, err := matchAny(rule.Exclude, file)
		if err != nil {
			return nil, fmt.Errorf("rule %q: exclude: %w", rule.Name, err)
		}
		if excluded {
			continue
		}
		matched = append(matched, file)
	}
	return matched, nil
}

func matchAny(patterns []string, file string) (bool, error) {
	for _, pattern := range patterns {
		ok, err := doublestar.Match(pattern, file)
		if err != nil {
			return false, fmt.Errorf("pattern %q is invalid: %w", pattern, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func lintRule(root string, rule config.Rule, files []string) []Issue {
	var issues []Issue
	docs := make([]document, 0, len(files))

	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			issues = append(issues, Issue{File: file, Message: fmt.Sprintf("cannot be read: %v", err)})
			continue
		}

		fields, err := frontmatter.ParseFile(string(data))
		if err != nil {
			issues = append(issues, Issue{File: file, Message: err.Error()})
			continue
		}

		doc, docIssues := lintDocument(rule, file, fields)
		issues = append(issues, docIssues...)
		docs = append(docs, doc)
	}

	issues = append(issues, lintUnique(rule, docs)...)
	issues = append(issues, lintReferences(rule, docs)...)
	return issues
}

func lintDocument(rule config.Rule, file string, fields map[string]yaml.Node) (document, []Issue) {
	doc := document{
		path:   file,
		values: map[string]string{},
		lists:  map[string][]string{},
	}
	var issues []Issue

	if !rule.AllowUnknownFields {
		var unknown []string
		for name := range fields {
			if _, ok := rule.Fields[name]; !ok {
				unknown = append(unknown, name)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			issues = append(issues, Issue{File: file, Message: fmt.Sprintf("unknown fields %s", quote(unknown))})
		}
	}

	for _, name := range rule.FieldNames() {
		field := rule.Fields[name]
		node, present := fields[name]
		if !present {
			if field.Required {
				issues = append(issues, Issue{File: file, Message: fmt.Sprintf("missing required field %q", name)})
			}
			continue
		}

		switch field.Type {
		case config.TypeString:
			value, err := scalarValue(node, name)
			if err != nil {
				issues = append(issues, Issue{File: file, Message: err.Error()})
				continue
			}
			issues = append(issues, checkString(file, name, field, value)...)
			doc.values[name] = value
		case config.TypeStringArray:
			values, err := sequenceValues(node, name)
			if err != nil {
				issues = append(issues, Issue{File: file, Message: err.Error()})
				continue
			}
			for _, value := range values {
				issues = append(issues, checkString(file, name, field, value)...)
			}
			doc.lists[name] = values
		}
	}

	if rule.FilenameField != "" {
		if value, ok := doc.values[rule.FilenameField]; ok {
			base := strings.TrimSuffix(filepath.Base(file), ".md")
			if base != value {
				issues = append(issues, Issue{
					File:    file,
					Message: fmt.Sprintf("filename %q does not match %s %q", base+".md", rule.FilenameField, value),
				})
			}
		}
	}

	return doc, issues
}

func checkString(file, name string, field config.Field, value string) []Issue {
	var issues []Issue
	if re := field.Regexp(); re != nil && !re.MatchString(value) {
		issues = append(issues, Issue{
			File:    file,
			Message: fmt.Sprintf("%s %q does not match pattern %s", name, value, field.Pattern),
		})
	}
	if len(field.Enum) > 0 && !contains(field.Enum, value) {
		issues = append(issues, Issue{
			File:    file,
			Message: fmt.Sprintf("%s %q is invalid (expected %s)", name, value, strings.Join(field.Enum, ", ")),
		})
	}
	return issues
}

func lintUnique(rule config.Rule, docs []document) []Issue {
	var issues []Issue
	for _, name := range rule.FieldNames() {
		if !rule.Fields[name].Unique {
			continue
		}
		valueToFiles := map[string][]string{}
		for _, doc := range docs {
			if value, ok := doc.values[name]; ok {
				valueToFiles[value] = append(valueToFiles[value], doc.path)
			}
		}
		for value, files := range valueToFiles {
			if len(files) < 2 {
				continue
			}
			sort.Strings(files)
			for _, file := range files {
				issues = append(issues, Issue{
					File:    file,
					Message: fmt.Sprintf("duplicate %s %q (also used in %s)", name, value, strings.Join(others(files, file), ", ")),
				})
			}
		}
	}
	return issues
}

func lintReferences(rule config.Rule, docs []document) []Issue {
	var issues []Issue
	for _, name := range rule.FieldNames() {
		field := rule.Fields[name]
		if field.References == "" {
			continue
		}

		known := map[string]struct{}{}
		for _, doc := range docs {
			if value, ok := doc.values[field.References]; ok {
				known[value] = struct{}{}
			}
		}

		for _, doc := range docs {
			own := doc.values[field.References]
			for _, ref := range doc.lists[name] {
				if ref == own && own != "" {
					if !field.SelfAllowed {
						issues = append(issues, Issue{
							File:    doc.path,
							Message: fmt.Sprintf("%s must not reference its own %s %q", name, field.References, ref),
						})
					}
					continue
				}
				if _, ok := known[ref]; !ok {
					issues = append(issues, Issue{
						File:    doc.path,
						Message: fmt.Sprintf("%s references missing %s %q", name, field.References, ref),
					})
				}
			}
		}

		if field.Acyclic {
			issues = append(issues, lintCycles(name, field.References, docs)...)
		}
	}
	return issues
}

func lintCycles(name, refField string, docs []document) []Issue {
	edges := map[string][]string{}
	pathOf := map[string]string{}
	for _, doc := range docs {
		id, ok := doc.values[refField]
		if !ok {
			continue
		}
		edges[id] = doc.lists[name]
		pathOf[id] = doc.path
	}

	const (
		unvisited = iota
		visiting
		done
	)
	color := map[string]int{}
	inCycle := map[string]string{}
	stack := make([]string, 0, len(edges))

	var visit func(id string)
	visit = func(id string) {
		color[id] = visiting
		stack = append(stack, id)

		for _, next := range edges[id] {
			if next == id {
				continue
			}
			if _, known := edges[next]; !known {
				continue
			}
			switch color[next] {
			case visiting:
				start := 0
				for i, node := range stack {
					if node == next {
						start = i
						break
					}
				}
				nodes := append(append([]string{}, stack[start:]...), next)
				cycle := strings.Join(nodes, " -> ")
				for _, node := range nodes {
					inCycle[node] = cycle
				}
			case unvisited:
				visit(next)
			}
		}

		color[id] = done
		stack = stack[:len(stack)-1]
	}

	ids := make([]string, 0, len(edges))
	for id := range edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == unvisited {
			visit(id)
		}
	}

	var issues []Issue
	for _, id := range ids {
		if cycle, ok := inCycle[id]; ok {
			issues = append(issues, Issue{
				File:    pathOf[id],
				Message: fmt.Sprintf("%s is part of a dependency cycle: %s", name, cycle),
			})
		}
	}
	return issues
}

func scalarValue(node yaml.Node, name string) (string, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return "", fmt.Errorf("%s must not be null", name)
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", fmt.Errorf("%s must be a string", name)
	}
	if node.Value == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return node.Value, nil
}

func sequenceValues(node yaml.Node, name string) ([]string, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}
	values := make([]string, 0, len(node.Content))
	for _, raw := range node.Content {
		item := frontmatter.Resolve(*raw)
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" || item.Value == "" {
			return nil, fmt.Errorf("%s must be an array of non-empty strings", name)
		}
		values = append(values, item.Value)
	}
	return values, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func others(files []string, current string) []string {
	rest := make([]string, 0, len(files)-1)
	for _, file := range files {
		if file != current {
			rest = append(rest, file)
		}
	}
	return rest
}

func quote(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return strings.Join(quoted, ", ")
}
