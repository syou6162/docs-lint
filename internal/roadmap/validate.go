package roadmap

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var validTypes = map[string]struct{}{
	"bug":           {},
	"refactoring":   {},
	"documentation": {},
	"test":          {},
	"feature":       {},
}

var validPriorities = map[string]struct{}{
	"high":   {},
	"medium": {},
	"low":    {},
}

// Issue is a single validation problem found in a roadmap task file.
type Issue struct {
	File    string
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s: %s", i.File, i.Message)
}

// ValidateDir validates all roadmap task files under dir.
func ValidateDir(dir string) ([]Issue, error) {
	tasks, issues, err := LoadTasks(dir)
	if err != nil {
		return nil, err
	}
	issues = append(issues, validateTasks(tasks)...)
	return issues, nil
}

// LoadTasks reads all roadmap task files under dir.
func LoadTasks(dir string) ([]Task, []Issue, error) {
	var tasks []Task
	var issues []Issue

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		if shouldSkipTaskFile(d.Name()) {
			return nil
		}

		task, err := parseTaskFile(path)
		if err != nil {
			issues = append(issues, Issue{File: path, Message: "parse: " + err.Error()})
			return nil
		}
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return tasks, issues, nil
}

func shouldSkipTaskFile(name string) bool {
	return name == "AGENTS.md" || name == "overview.md"
}

func validateTasks(tasks []Task) []Issue {
	var issues []Issue

	idToFiles := map[string][]string{}
	knownIDs := map[string]struct{}{}

	for _, task := range tasks {
		idToFiles[task.ID] = append(idToFiles[task.ID], task.Path)
		knownIDs[task.ID] = struct{}{}
	}

	for id, files := range idToFiles {
		if len(files) < 2 {
			continue
		}
		for _, file := range files {
			issues = append(issues, Issue{
				File:    file,
				Message: fmt.Sprintf("validate: duplicate id %q (also used in %s)", id, joinOtherFiles(files, file)),
			})
		}
	}

	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			if dep == task.ID {
				issues = append(issues, Issue{
					File:    task.Path,
					Message: fmt.Sprintf("validate: depends_on must not reference its own id %q", dep),
				})
				continue
			}
			if _, ok := knownIDs[dep]; ok {
				continue
			}
			issues = append(issues, Issue{
				File:    task.Path,
				Message: fmt.Sprintf("validate: depends_on references missing task id %q", dep),
			})
		}
	}

	dependencyCycles := findDependencyCycles(tasks)

	for _, task := range tasks {
		if cycle, ok := dependencyCycles[task.ID]; ok {
			issues = append(issues, Issue{
				File:    task.Path,
				Message: fmt.Sprintf("validate: depends_on is part of a dependency cycle: %s", cycle),
			})
		}
	}

	return issues
}

func findDependencyCycles(tasks []Task) map[string]string {
	deps := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		deps[task.ID] = task.DependsOn
	}

	const (
		unvisited = iota
		visiting
		done
	)

	color := make(map[string]int, len(tasks))
	inCycle := make(map[string]string)
	path := make([]string, 0, len(tasks))

	var dfs func(id string)
	dfs = func(id string) {
		color[id] = visiting
		path = append(path, id)

		for _, dep := range deps[id] {
			if dep == id {
				continue
			}
			if _, known := deps[dep]; !known {
				continue
			}
			switch color[dep] {
			case visiting:
				start := 0
				for i, node := range path {
					if node == dep {
						start = i
						break
					}
				}
				cycleNodes := append(append([]string{}, path[start:]...), dep)
				cycle := strings.Join(cycleNodes, " -> ")
				for _, node := range cycleNodes {
					inCycle[node] = cycle
				}
			case unvisited:
				dfs(dep)
			}
		}

		color[id] = done
		path = path[:len(path)-1]
	}

	for id := range deps {
		if color[id] == unvisited {
			dfs(id)
		}
	}

	return inCycle
}

func joinOtherFiles(files []string, current string) string {
	others := make([]string, 0, len(files)-1)
	for _, file := range files {
		if file == current {
			continue
		}
		others = append(others, file)
	}
	return strings.Join(others, ", ")
}
