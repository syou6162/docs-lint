package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/syou6162/docs-lint/internal/roadmap"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: docs-lint <subcommand> [args]\n\nsubcommands:\n  validate [dir]  validate roadmap task files\n  tasks [dir]     list available roadmap tasks\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		runValidate(os.Args[2:])
	case "tasks":
		runTasks(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func runValidate(args []string) {
	dir := "docs/roadmap"
	if len(args) > 0 {
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

func runTasks(args []string) {
	fs := flag.NewFlagSet("tasks", flag.ExitOnError)
	priority := fs.String("priority", "", "filter by priority (high, medium, low)")
	taskType := fs.String("type", "", "filter by type (bug, refactoring, documentation, test, feature)")
	sortBy := fs.String("sort", "priority", "sort by priority or type")
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: roadmap tasks [flags] [dir]\n\nflags:\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	dir := "docs/roadmap"
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	opts := roadmap.ListTasksOptions{
		Priority: *priority,
		Type:     *taskType,
		Sort:     *sortBy,
	}

	tasks, err := roadmap.ListAvailableTasks(dir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs-lint: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, err := roadmap.FormatTasksJSON(tasks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "docs-lint: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	fmt.Print(roadmap.FormatTasksTable(tasks))
}
