package roadmap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ListTasksOptions configures filtering and sorting for available tasks.
type ListTasksOptions struct {
	Priority string
	Type     string
	Sort     string
}

var priorityRank = map[string]int{
	"high":   3,
	"medium": 2,
	"low":    1,
}

// ParseListTasksOptions validates listing/filtering options at the input boundary.
func ParseListTasksOptions(opts ListTasksOptions) (ListTasksOptions, error) {
	if opts.Priority != "" {
		if _, ok := validPriorities[opts.Priority]; !ok {
			return ListTasksOptions{}, fmt.Errorf("invalid --priority %q (expected high, medium, or low)", opts.Priority)
		}
	}
	if opts.Type != "" {
		if _, ok := validTypes[opts.Type]; !ok {
			return ListTasksOptions{}, fmt.Errorf("invalid --type %q (expected bug, refactoring, documentation, test, or feature)", opts.Type)
		}
	}

	sortBy := opts.Sort
	if sortBy == "" {
		sortBy = "priority"
	}
	switch sortBy {
	case "priority", "type":
		opts.Sort = sortBy
		return opts, nil
	default:
		return ListTasksOptions{}, fmt.Errorf("invalid --sort %q (expected priority or type)", sortBy)
	}
}

// ListAvailableTasks returns tasks whose dependencies are all completed,
// after applying optional filters and sorting.
func ListAvailableTasks(dir string, opts ListTasksOptions) ([]Task, error) {
	opts, err := ParseListTasksOptions(opts)
	if err != nil {
		return nil, err
	}

	tasks, issues, err := LoadTasks(dir)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		msgs := make([]string, len(issues))
		for i, issue := range issues {
			msgs[i] = issue.String()
		}
		return nil, fmt.Errorf("%s", strings.Join(msgs, "\n"))
	}

	available := filterAvailable(tasks)
	available = filterByPriority(available, opts.Priority)
	available = filterByType(available, opts.Type)
	if err := sortTasks(available, opts.Sort); err != nil {
		return nil, err
	}

	return available, nil
}

func filterAvailable(tasks []Task) []Task {
	knownIDs := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		knownIDs[task.ID] = struct{}{}
	}

	available := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if isAvailable(task, knownIDs) {
			available = append(available, task)
		}
	}
	return available
}

// isAvailable reports whether all dependencies are completed.
// A dependency is completed when its task id is no longer present in the repo.
func isAvailable(task Task, knownIDs map[string]struct{}) bool {
	for _, dep := range task.DependsOn {
		if _, ok := knownIDs[dep]; ok {
			return false
		}
	}
	return true
}

func filterByPriority(tasks []Task, priority string) []Task {
	if priority == "" {
		return tasks
	}
	filtered := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Priority == priority {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func filterByType(tasks []Task, typ string) []Task {
	if typ == "" {
		return tasks
	}
	filtered := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Type == typ {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func sortTasks(tasks []Task, sortBy string) error {
	switch sortBy {
	case "type":
		sort.Slice(tasks, func(i, j int) bool {
			if tasks[i].Type != tasks[j].Type {
				return tasks[i].Type < tasks[j].Type
			}
			return tasks[i].ID < tasks[j].ID
		})
	case "priority":
		sort.Slice(tasks, func(i, j int) bool {
			pi := priorityRank[tasks[i].Priority]
			pj := priorityRank[tasks[j].Priority]
			if pi != pj {
				return pi > pj
			}
			return tasks[i].ID < tasks[j].ID
		})
	default:
		return fmt.Errorf("invalid --sort %q (expected priority or type)", sortBy)
	}
	return nil
}

type taskJSON struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Path     string `json:"path"`
}

// FormatTasksJSON returns a JSON representation of tasks.
func FormatTasksJSON(tasks []Task) ([]byte, error) {
	items := make([]taskJSON, len(tasks))
	for i, task := range tasks {
		items[i] = taskJSON{
			ID:       task.ID,
			Title:    task.Title,
			Type:     task.Type,
			Priority: task.Priority,
			Path:     task.Path,
		}
	}
	return json.MarshalIndent(items, "", "  ")
}

// FormatTasksTable returns a column-aligned table of tasks.
func FormatTasksTable(tasks []Task) string {
	idWidth := len("ID")
	typeWidth := len("TYPE")
	priorityWidth := len("PRIORITY")

	for _, task := range tasks {
		idWidth = max(idWidth, len(task.ID))
		typeWidth = max(typeWidth, len(task.Type))
		priorityWidth = max(priorityWidth, len(task.Priority))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %s\n", idWidth, "ID", typeWidth, "TYPE", priorityWidth, "PRIORITY", "TITLE")
	for _, task := range tasks {
		fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %s\n", idWidth, task.ID, typeWidth, task.Type, priorityWidth, task.Priority, task.Title)
	}
	return b.String()
}
