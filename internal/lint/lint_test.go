package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syou6162/docs-lint/internal/config"
)

const roadmapConfig = `
rules:
  - name: roadmap-task
    include:
      - "docs/roadmap/**/*.md"
    exclude:
      - "**/AGENTS.md"
      - "**/overview.md"
    filename_field: id
    fields:
      id:
        type: string
        required: true
        pattern: "^[a-z0-9]+(-[a-z0-9]+)*$"
        unique: true
      title:
        type: string
        required: true
      priority:
        type: string
        required: true
        enum: [high, medium, low]
      depends_on:
        type: string_array
        required: true
        references: id
        acyclic: true
`

func TestRunItemNameInReferenceMessage(t *testing.T) {
	cfg := strings.Replace(roadmapConfig, "    filename_field: id\n", "    filename_field: id\n    item_name: task\n", 1)
	issues := run(t, cfg, map[string]string{
		"docs/roadmap/backlog/a.md": task("a", "t", "high", "[nope]"),
	})
	want := `docs/roadmap/backlog/a.md: validate: depends_on references missing task id "nope"`
	if len(issues) != 1 || issues[0].String() != want {
		t.Errorf("Run() issues = %v, want [%s]", issues, want)
	}
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func task(id, title, priority, dependsOn string) string {
	return "---\nid: " + id + "\ntitle: " + title + "\npriority: " + priority + "\ndepends_on: " + dependsOn + "\n---\n\n# body\n"
}

func run(t *testing.T, cfgYAML string, files map[string]string) []Issue {
	t.Helper()
	cfg, err := config.Parse([]byte(cfgYAML))
	if err != nil {
		t.Fatalf("config.Parse() error = %v", err)
	}
	issues, err := Run(writeFiles(t, files), cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return issues
}

func TestRunValid(t *testing.T) {
	issues := run(t, roadmapConfig, map[string]string{
		"docs/roadmap/backlog/first-task.md":  task("first-task", "first", "high", "[]"),
		"docs/roadmap/backlog/second-task.md": task("second-task", "second", "low", "[first-task]"),
		// Excluded and out-of-scope files must not be linted.
		"docs/roadmap/AGENTS.md":         "# not a task\n",
		"docs/roadmap/theme/overview.md": "# not a task\n",
		"docs/requirements/overview.md":  "# unrelated\n",
		"README.md":                      "# unrelated\n",
	})
	if len(issues) != 0 {
		t.Fatalf("Run() issues = %v, want none", issues)
	}
}

func TestRunReportsFieldProblems(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		want    string
	}{
		{
			name:    "missing front-matter",
			file:    "docs/roadmap/backlog/a.md",
			content: "# body\n",
			want:    "docs/roadmap/backlog/a.md: parse: missing YAML front-matter",
		},
		{
			name:    "missing required field",
			file:    "docs/roadmap/backlog/a.md",
			content: "---\nid: a\ntitle: t\npriority: high\n---\n",
			want:    `docs/roadmap/backlog/a.md: parse: missing required field "depends_on"`,
		},
		{
			name:    "unknown field",
			file:    "docs/roadmap/backlog/a.md",
			content: "---\nid: a\ntitle: t\npriority: high\ndepends_on: []\nowner: me\n---\n",
			want:    `docs/roadmap/backlog/a.md: parse: unknown fields "owner"`,
		},
		{
			name:    "pattern violation",
			file:    "docs/roadmap/backlog/Bad_ID.md",
			content: task("Bad_ID", "t", "high", "[]"),
			want:    `docs/roadmap/backlog/Bad_ID.md: parse: id "Bad_ID" does not match required pattern ^[a-z0-9]+(-[a-z0-9]+)*$`,
		},
		{
			name:    "enum violation",
			file:    "docs/roadmap/backlog/a.md",
			content: task("a", "t", "urgent", "[]"),
			want:    `docs/roadmap/backlog/a.md: parse: priority "urgent" is invalid (expected high, medium, or low)`,
		},
		{
			name:    "filename mismatch",
			file:    "docs/roadmap/backlog/other.md",
			content: task("a", "t", "high", "[]"),
			want:    `docs/roadmap/backlog/other.md: parse: filename "other.md" does not match id "a"`,
		},
		{
			name:    "null value",
			file:    "docs/roadmap/backlog/a.md",
			content: "---\nid: a\ntitle:\npriority: high\ndepends_on: []\n---\n",
			want:    "docs/roadmap/backlog/a.md: parse: title must not be null",
		},
		{
			name:    "wrong scalar type",
			file:    "docs/roadmap/backlog/a.md",
			content: "---\nid: a\ntitle: 42\npriority: high\ndepends_on: []\n---\n",
			want:    "docs/roadmap/backlog/a.md: parse: title must be a string",
		},
		{
			name:    "array expected",
			file:    "docs/roadmap/backlog/a.md",
			content: task("a", "t", "high", "first-task"),
			want:    "docs/roadmap/backlog/a.md: parse: depends_on must be an array",
		},
		{
			name:    "missing reference",
			file:    "docs/roadmap/backlog/a.md",
			content: task("a", "t", "high", "[nope]"),
			want:    `docs/roadmap/backlog/a.md: validate: depends_on references missing id "nope"`,
		},
		{
			name:    "self reference",
			file:    "docs/roadmap/backlog/a.md",
			content: task("a", "t", "high", "[a]"),
			want:    `docs/roadmap/backlog/a.md: validate: depends_on must not reference its own id "a"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := run(t, roadmapConfig, map[string]string{test.file: test.content})
			if len(issues) != 1 {
				t.Fatalf("Run() issues = %v, want exactly 1", issues)
			}
			if got := issues[0].String(); got != test.want {
				t.Errorf("Run() issue = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRunDuplicateID(t *testing.T) {
	issues := run(t, roadmapConfig, map[string]string{
		"docs/roadmap/backlog/a.md":     task("a", "t", "high", "[]"),
		"docs/roadmap/theme/tasks/a.md": task("a", "t", "high", "[]"),
	})
	want := []string{
		`docs/roadmap/backlog/a.md: validate: duplicate id "a" (also used in docs/roadmap/theme/tasks/a.md)`,
		`docs/roadmap/theme/tasks/a.md: validate: duplicate id "a" (also used in docs/roadmap/backlog/a.md)`,
	}
	assertIssues(t, issues, want)
}

func TestRunDependencyCycle(t *testing.T) {
	issues := run(t, roadmapConfig, map[string]string{
		"docs/roadmap/backlog/a.md": task("a", "t", "high", "[b]"),
		"docs/roadmap/backlog/b.md": task("b", "t", "high", "[a]"),
	})
	want := []string{
		"docs/roadmap/backlog/a.md: validate: depends_on is part of a dependency cycle: a -> b -> a",
		"docs/roadmap/backlog/b.md: validate: depends_on is part of a dependency cycle: a -> b -> a",
	}
	assertIssues(t, issues, want)
}

func TestRunSelfReferenceAllowed(t *testing.T) {
	cfgYAML := `
rules:
  - name: page
    include: ["**/*.md"]
    fields:
      id:
        type: string
        required: true
      see_also:
        type: string_array
        references: id
        self_reference_allowed: true
`
	issues := run(t, cfgYAML, map[string]string{
		"a.md": "---\nid: a\nsee_also: [a]\n---\n",
	})
	if len(issues) != 0 {
		t.Fatalf("Run() issues = %v, want none", issues)
	}
}

func TestRunAllowUnknownFields(t *testing.T) {
	cfgYAML := `
rules:
  - name: page
    include: ["**/*.md"]
    allow_unknown_fields: true
    fields:
      id:
        type: string
        required: true
`
	issues := run(t, cfgYAML, map[string]string{
		"a.md": "---\nid: a\nowner: me\n---\n",
	})
	if len(issues) != 0 {
		t.Fatalf("Run() issues = %v, want none", issues)
	}
}

func TestRunMissingRoot(t *testing.T) {
	cfg, err := config.Parse([]byte(roadmapConfig))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(filepath.Join(t.TempDir(), "nope"), cfg); err == nil {
		t.Fatal("Run() error = nil, want error for a missing directory")
	}
}

func assertIssues(t *testing.T, issues []Issue, want []string) {
	t.Helper()
	if len(issues) != len(want) {
		t.Fatalf("Run() issues = %v, want %d issues", issues, len(want))
	}
	for i, issue := range issues {
		if got := issue.String(); got != want[i] {
			t.Errorf("issue[%d] = %q, want %q", i, got, want[i])
		}
	}
}
