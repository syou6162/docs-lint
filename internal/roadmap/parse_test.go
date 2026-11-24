package roadmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTaskFileRejects(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		filename     string
		wantContains []string
	}{
		{
			name: "depends_on null",
			content: `---
id: task-a
title: example
type: feature
priority: high
depends_on: null
---
`,
			filename:     "task-a.md",
			wantContains: []string{"depends_on must be an array"},
		},
		{
			name: "unknown field",
			content: `---
id: task-b
title: example
type: feature
priority: high
depends_on: []
foo: bar
---
`,
			filename:     "task-b.md",
			wantContains: []string{`unknown fields "foo"`},
		},
		{
			name: "invalid decision scope",
			content: `---
id: task-decision-scope
title: example
type: feature
priority: high
depends_on: []
decision_scope: unresolved
---
`,
			filename:     "task-decision-scope.md",
			wantContains: []string{`decision_scope "unresolved" is invalid`},
		},
		{
			name:         "empty decision scope",
			content:      "---\nid: task-decision-scope-empty\ntitle: example\ntype: feature\npriority: high\ndepends_on: []\ndecision_scope: \"\"\n---\n",
			filename:     "task-decision-scope-empty.md",
			wantContains: []string{"decision_scope must be a non-empty string"},
		},
		{
			name:         "null decision scope",
			content:      "---\nid: task-decision-scope-null\ntitle: example\ntype: feature\npriority: high\ndepends_on: []\ndecision_scope: null\n---\n",
			filename:     "task-decision-scope-null.md",
			wantContains: []string{"decision_scope must not be null"},
		},
		{
			name: "unknown fields sorted",
			content: `---
id: task-b2
title: example
type: feature
priority: high
depends_on: []
zebra: 1
alpha: 2
---
`,
			filename:     "task-b2.md",
			wantContains: []string{`unknown fields "alpha", "zebra"`},
		},
		{
			name: "empty id scalar",
			content: `---
id: ""
title: example
type: feature
priority: high
depends_on: []
---
`,
			filename:     "task-c.md",
			wantContains: []string{"id must be a non-empty string"},
		},
		{
			name: "null id",
			content: `---
id: null
title: example
type: feature
priority: high
depends_on: []
---
`,
			filename:     "task-c2.md",
			wantContains: []string{"id must not be null"},
		},
		{
			name: "null title",
			content: `---
id: task-d
title: null
type: feature
priority: high
depends_on: []
---
`,
			filename:     "task-d.md",
			wantContains: []string{"title must not be null"},
		},
		{
			name: "null type",
			content: `---
id: task-d2
title: example
type: null
priority: high
depends_on: []
---
`,
			filename:     "task-d2.md",
			wantContains: []string{"type must not be null"},
		},
		{
			name: "null priority",
			content: `---
id: task-d3
title: example
type: feature
priority: null
depends_on: []
---
`,
			filename:     "task-d3.md",
			wantContains: []string{"priority must not be null"},
		},
		{
			name: "invalid type",
			content: `---
id: task-e
title: example
type: chore
priority: high
depends_on: []
---
`,
			filename:     "task-e.md",
			wantContains: []string{`type "chore" is invalid`},
		},
		{
			name: "invalid priority",
			content: `---
id: task-f
title: example
type: feature
priority: urgent
depends_on: []
---
`,
			filename:     "task-f.md",
			wantContains: []string{`priority "urgent" is invalid`},
		},
		{
			name: "filename mismatch",
			content: `---
id: task-g
title: example
type: feature
priority: high
depends_on: []
---
`,
			filename:     "other-name.md",
			wantContains: []string{`does not match id`},
		},
		{
			name: "empty dependency item",
			content: `---
id: task-h
title: example
type: feature
priority: high
depends_on:
  - ""
---
`,
			filename:     "task-h.md",
			wantContains: []string{"depends_on must be an array of id strings"},
		},
		{
			name: "invalid dependency id format",
			content: `---
id: task-i
title: example
type: feature
priority: high
depends_on:
  - Bad_ID
---
`,
			filename:     "task-i.md",
			wantContains: []string{`depends_on item "Bad_ID" does not match required pattern`},
		},
		{
			name: "integer id",
			content: `---
id: 123
title: example
type: feature
priority: high
depends_on: []
---
`,
			filename:     "123.md",
			wantContains: []string{"id must be a string"},
		},
		{
			name: "boolean id",
			content: `---
id: true
title: example
type: feature
priority: high
depends_on: []
---
`,
			filename:     "true.md",
			wantContains: []string{"id must be a string"},
		},
		{
			name: "integer depends_on item",
			content: `---
id: task-j
title: example
type: feature
priority: high
depends_on:
  - 1
---
`,
			filename:     "task-j.md",
			wantContains: []string{"depends_on must be an array of id strings"},
		},
		{
			name: "depends_on null item",
			content: `---
id: task-k
title: example
type: feature
priority: high
depends_on:
  - null
---
`,
			filename:     "task-k.md",
			wantContains: []string{"depends_on must be an array of id strings"},
		},
		{
			name: "non-mapping front matter",
			content: `---
[]
---
`,
			filename:     "task-l.md",
			wantContains: []string{`missing required field "id"`},
		},
		{
			name: "type sequence",
			content: `---
id: task-m
title: example
type: [feature]
priority: high
depends_on: []
---
`,
			filename:     "task-m.md",
			wantContains: []string{"type must be a string"},
		},
		{
			name: "duplicate field",
			content: `---
id: task-n
title: first
title: second
type: feature
priority: high
depends_on: []
---
`,
			filename:     "task-n.md",
			wantContains: []string{`duplicate field "title"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.filename)
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			_, err := parseTaskFile(path)
			if err == nil {
				t.Fatal("parseTaskFile() error = nil, want parse error")
			}

			msg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Fatalf("error = %q, want substring %q", msg, want)
				}
			}
		})
	}
}

func TestParseTaskFileAcceptsWhitespaceOnlyTitle(t *testing.T) {
	content := `---
id: ws-title
title: '   '
type: feature
priority: high
depends_on: []
---
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ws-title.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	task, err := parseTaskFile(path)
	if err != nil {
		t.Fatalf("parseTaskFile() error = %v, want success for whitespace-only title", err)
	}
	if task.Title != "   " {
		t.Fatalf("task.Title = %q, want %q", task.Title, "   ")
	}
}

func TestParseTaskFileAcceptsOptionalDecisionScope(t *testing.T) {
	tests := []struct {
		name          string
		decisionScope string
	}{
		{name: "omitted"},
		{name: "ready", decisionScope: "ready"},
		{name: "implementation", decisionScope: "implementation"},
		{name: "design", decisionScope: "design"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `---
id: decision-scope
title: example
type: feature
priority: high
depends_on: []
`
			if tt.decisionScope != "" {
				content += "decision_scope: " + tt.decisionScope + "\n"
			}
			content += "---\n"

			path := filepath.Join(t.TempDir(), "decision-scope.md")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			task, err := parseTaskFile(path)
			if err != nil {
				t.Fatalf("parseTaskFile() error = %v, want success", err)
			}
			if task.DecisionScope != tt.decisionScope {
				t.Fatalf("task.DecisionScope = %q, want %q", task.DecisionScope, tt.decisionScope)
			}
		})
	}
}
