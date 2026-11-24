package roadmap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syou6162/docs-lint/internal/roadmap"
)

func TestValidateDirValid(t *testing.T) {
	dir := filepath.Join("testdata", "valid")
	issues, err := roadmap.ValidateDir(dir)
	if err != nil {
		t.Fatalf("ValidateDir() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none", issues)
	}
}

func TestValidateDirDecisionScopeRequirement(t *testing.T) {
	dir := t.TempDir()
	content := "---\nid: task-without-decision-scope\ntitle: example\ntype: feature\npriority: high\ndepends_on: []\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "task-without-decision-scope.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tests := []struct {
		name                  string
		options               roadmap.ValidateOptions
		wantMissingScopeIssue bool
	}{
		{name: "optional by default"},
		{
			name:                  "required when configured",
			options:               roadmap.ValidateOptions{RequireDecisionScope: true},
			wantMissingScopeIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := roadmap.ValidateDirWithOptions(dir, tt.options)
			if err != nil {
				t.Fatalf("ValidateDirWithOptions() error = %v", err)
			}
			joined := joinIssues(issues)
			if tt.wantMissingScopeIssue {
				if !strings.Contains(joined, `missing required field "decision_scope"`) {
					t.Fatalf("issues = %q, want missing decision_scope", joined)
				}
				return
			}
			if len(issues) != 0 {
				t.Fatalf("issues = %#v, want none", issues)
			}
		})
	}
}

func TestValidateDirInvalidCases(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		wantContains []string
	}{
		{
			name: "bad id format",
			dir:  filepath.Join("testdata", "invalid-cases", "bad-id-format"),
			wantContains: []string{
				"parse:",
				`does not match required pattern`,
				"bad-id-format.md",
			},
		},
		{
			name: "filename mismatch",
			dir:  filepath.Join("testdata", "invalid-cases", "filename-mismatch"),
			wantContains: []string{
				"parse:",
				`does not match id`,
				"filename-mismatch.md",
			},
		},
		{
			name: "missing dependency",
			dir:  filepath.Join("testdata", "invalid-cases", "missing-dependency"),
			wantContains: []string{
				"validate:",
				`depends_on references missing task id`,
				"missing-dependency.md",
			},
		},
		{
			name: "bad type",
			dir:  filepath.Join("testdata", "invalid-cases", "bad-type"),
			wantContains: []string{
				"parse:",
				`type "chore" is invalid`,
				"bad-type.md",
			},
		},
		{
			name: "bad priority",
			dir:  filepath.Join("testdata", "invalid-cases", "bad-priority"),
			wantContains: []string{
				"parse:",
				`priority "urgent" is invalid`,
				"bad-priority.md",
			},
		},
		{
			name: "empty title",
			dir:  filepath.Join("testdata", "invalid-cases", "empty-title"),
			wantContains: []string{
				"parse:",
				`title must be a non-empty string`,
				"empty-title.md",
			},
		},
		{
			name: "missing field",
			dir:  filepath.Join("testdata", "invalid-cases", "missing-field"),
			wantContains: []string{
				"parse:",
				`missing required field "depends_on"`,
				"missing-field.md",
			},
		},
		{
			name: "unknown field",
			dir:  filepath.Join("testdata", "invalid-cases", "unknown-field"),
			wantContains: []string{
				"parse:",
				`unknown fields "extra"`,
				"unknown-field.md",
			},
		},
		{
			name: "empty dependency item",
			dir:  filepath.Join("testdata", "invalid-cases", "empty-dependency-item"),
			wantContains: []string{
				"parse:",
				"depends_on must be an array of id strings",
				"empty-dependency-item.md",
			},
		},
		{
			name: "duplicate id",
			dir:  filepath.Join("testdata", "duplicate-id"),
			wantContains: []string{
				"validate:",
				`duplicate id "duplicate-task"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := roadmap.ValidateDir(tt.dir)
			if err != nil {
				t.Fatalf("ValidateDir() error = %v", err)
			}
			if len(issues) == 0 {
				t.Fatal("issues is empty, want validation errors")
			}

			joined := joinIssues(issues)
			for _, want := range tt.wantContains {
				if !strings.Contains(joined, want) {
					t.Fatalf("issues = %q, want substring %q", joined, want)
				}
			}
		})
	}
}

func TestShouldSkipTaskFiles(t *testing.T) {
	dir := filepath.Join("testdata", "skipped")
	issues, err := roadmap.ValidateDir(dir)
	if err != nil {
		t.Fatalf("ValidateDir() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none", issues)
	}
}

func joinIssues(issues []roadmap.Issue) string {
	var b strings.Builder
	for i, issue := range issues {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(issue.String())
	}
	return b.String()
}
