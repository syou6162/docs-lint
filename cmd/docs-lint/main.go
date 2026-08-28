// Command docs-lint validates the YAML front-matter of Markdown files against
// rules declared in a config file.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/syou6162/docs-lint/internal/config"
	"github.com/syou6162/docs-lint/internal/lint"
)

// defaultConfigPaths are tried in order when -config is not given.
var defaultConfigPaths = []string{"docs-lint.yaml", "docs-lint.yml", ".docs-lint.yaml", ".docs-lint.yml"}

// Exit codes: violations and usage/IO errors are distinguished so that CI can
// tell a failing document from a broken invocation.
const (
	exitOK        = 0
	exitViolation = 1
	exitError     = 2
)

func main() {
	configPath := flag.String("config", "", "path to the config file (default: docs-lint.yaml)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: docs-lint [flags] [dir]\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "docs-lint: expected at most one directory, got %d arguments\n", flag.NArg())
		flag.Usage()
		os.Exit(exitError)
	}
	root := "."
	if flag.NArg() == 1 {
		root = flag.Arg(0)
	}

	os.Exit(run(*configPath, root, os.Stdout, os.Stderr))
}

func run(configPath, root string, stdout, stderr io.Writer) int {
	path, err := resolveConfigPath(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "docs-lint: %v\n", err)
		return exitError
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "docs-lint: %v\n", err)
		return exitError
	}

	issues, err := lint.Run(root, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "docs-lint: %v\n", err)
		return exitError
	}

	for _, issue := range issues {
		fmt.Fprintln(stdout, issue)
	}
	if len(issues) > 0 {
		return exitViolation
	}
	return exitOK
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	for _, candidate := range defaultConfigPaths {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no config file found (looked for %v); pass -config", defaultConfigPaths)
}
