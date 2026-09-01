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

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
