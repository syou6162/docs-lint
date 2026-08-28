package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
