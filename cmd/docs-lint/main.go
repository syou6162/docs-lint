package main

import (
	"fmt"
	"os"

	"github.com/syou6162/docs-lint/internal/roadmap"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: docs-lint [dir]\n\nvalidate roadmap task files (default dir: docs/roadmap)\n")
}

func main() {
	args := os.Args[1:]
	if len(args) > 1 {
		usage()
		os.Exit(1)
	}

	dir := "docs/roadmap"
	if len(args) == 1 {
		dir = args[0]
	}

	issues, err := roadmap.ValidateDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs-lint: %v\n", err)
		os.Exit(1)
	}

	for _, issue := range issues {
		fmt.Println(issue)
	}
	if len(issues) > 0 {
		os.Exit(1)
	}
}
