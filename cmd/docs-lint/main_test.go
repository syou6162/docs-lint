package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTasksRejectsUnknownSortFlag(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	taskDir := filepath.Join(wd, "..", "..", "internal", "roadmap", "testdata", "tasks")
	cmd := exec.Command("go", "run", ".", "tasks", "--sort", "title", taskDir)
	cmd.Dir = wd

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("roadmap tasks --sort title succeeded, want failure; output = %s", out)
	}
	if !strings.Contains(string(out), `invalid --sort "title"`) {
		t.Fatalf("output = %q, want invalid --sort message", string(out))
	}
}

func TestRunValidateDecisionScopeFlag(t *testing.T) {
	dir := t.TempDir()
	content := "---\nid: task-without-decision-scope\ntitle: example\ntype: feature\npriority: high\ndepends_on: []\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "task-without-decision-scope.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if code := runValidate([]string{dir}); code != 0 {
		t.Fatalf("runValidate() without required flag = %d, want 0", code)
	}
	if code := runValidate([]string{"--require-decision-scope", dir}); code != 1 {
		t.Fatalf("runValidate() with required flag = %d, want 1", code)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
