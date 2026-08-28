// Command docs-lint validates the YAML front-matter of Markdown files against
// rules declared in a config file.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/syou6162/docs-lint/internal/config"
	"github.com/syou6162/docs-lint/internal/lint"
)

// defaultConfigPaths are tried in order when -config is not given.
var defaultConfigPaths = []string{"docs-lint.yaml", "docs-lint.yml", ".docs-lint.yaml"}

func main() {
	configPath := flag.String("config", "", "path to the config file (default: docs-lint.yaml)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: docs-lint [flags] [dir]\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	if err := run(*configPath, root); err != nil {
		fmt.Fprintf(os.Stderr, "docs-lint: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath, root string) error {
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	issues, err := lint.Run(root, cfg)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		fmt.Println(issue)
	}
	if len(issues) > 0 {
		os.Exit(1)
	}
	return nil
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
