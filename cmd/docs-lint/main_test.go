package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `rules:
  - name: doc
    include:
      - "**/*.md"
    filename_field: id
    fields:
      id:
        type: string
        required: true
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		docs     map[string]string
		wantCode int
		wantOut  string
	}{
		{
			name:     "clean",
			config:   validConfig,
			docs:     map[string]string{"ok.md": "---\nid: ok\n---\n"},
			wantCode: exitOK,
		},
		{
			name:     "violation",
			config:   validConfig,
			docs:     map[string]string{"ok.md": "---\nid: other\n---\n"},
			wantCode: exitViolation,
			wantOut:  `ok.md: filename "ok.md" does not match id "other"`,
		},
		{
			name:     "invalid config",
			config:   "rules:\n  - name: doc\n    include: []\n",
			docs:     map[string]string{"ok.md": "---\nid: ok\n---\n"},
			wantCode: exitError,
		},
		{
			name:     "no file matches the rule",
			config:   validConfig,
			docs:     map[string]string{"ok.txt": "not markdown"},
			wantCode: exitViolation,
			wantOut:  `rule "doc": no Markdown file matched include`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "docs-lint.yaml"), tt.config)
			for name, content := range tt.docs {
				writeFile(t, filepath.Join(dir, name), content)
			}
			t.Chdir(dir)

			var stdout, stderr bytes.Buffer
			got := run("", ".", &stdout, &stderr)
			if got != tt.wantCode {
				t.Errorf("run() = %d, want %d (stdout %q, stderr %q)", got, tt.wantCode, stdout.String(), stderr.String())
			}
			if tt.wantOut != "" && !strings.Contains(stdout.String(), tt.wantOut) {
				t.Errorf("run() stdout = %q, want it to contain %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestRunMissingConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	if got := run("", ".", &stdout, &stderr); got != exitError {
		t.Errorf("run() = %d, want %d", got, exitError)
	}
	if !strings.Contains(stderr.String(), "no config file found") {
		t.Errorf("run() stderr = %q, want it to mention that no config was found", stderr.String())
	}
}

func TestResolveConfigPathExplicit(t *testing.T) {
	got, err := resolveConfigPath("some/other.yaml")
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}
	if want := "some/other.yaml"; got != want {
		t.Errorf("resolveConfigPath() = %q, want %q", got, want)
	}
}

func TestResolveConfigPathDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docs-lint.yaml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := resolveConfigPath("")
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}
	if want := "docs-lint.yaml"; got != want {
		t.Errorf("resolveConfigPath() = %q, want %q", got, want)
	}
}

func TestResolveConfigPathMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := resolveConfigPath("")
	if err == nil {
		t.Fatal("resolveConfigPath() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "no config file found") {
		t.Errorf("resolveConfigPath() error = %v, want it to mention that no config was found", err)
	}
}
