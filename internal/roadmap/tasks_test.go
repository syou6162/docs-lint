package roadmap_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syou6162/docs-lint/internal/roadmap"
)

func TestListAvailableTasks(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	got := taskIDs(tasks)
	want := []string{"available-high", "available-medium", "available-low"}
	if len(got) != len(want) {
		t.Fatalf("got %d tasks %v, want %d %v", len(got), got, len(want), want)
	}
	for i, id := range want {
		if got[i] != id {
			t.Fatalf("task[%d] = %q, want %q (all = %v)", i, got[i], id, got)
		}
	}
}

func TestListAvailableTasksFilterPriority(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	got := taskIDs(tasks)
	want := []string{"available-high"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListAvailableTasksFilterType(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{
		Type: "bug",
	})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	got := taskIDs(tasks)
	want := []string{"available-low"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListAvailableTasksSortByType(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{
		Sort: "type",
	})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	got := taskIDs(tasks)
	want := []string{"available-low", "available-high", "available-medium"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListAvailableTasksRejectsUnknownSort(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	_, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{
		Sort: "title",
	})
	if err == nil {
		t.Fatal("ListAvailableTasks() error = nil, want unknown sort error")
	}
	if !strings.Contains(err.Error(), `invalid --sort "title"`) {
		t.Fatalf("error = %q, want invalid --sort message", err.Error())
	}
}

func TestListAvailableTasksBlocksExistingDependency(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	for _, task := range tasks {
		if task.ID == "blocked-task" {
			t.Fatalf("blocked-task must be unavailable while dependency exists in repo")
		}
	}
}

func TestFormatTasksTable(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	table := roadmap.FormatTasksTable(tasks)
	if !strings.Contains(table, "ID") || !strings.Contains(table, "TYPE") {
		t.Fatalf("table missing headers: %q", table)
	}
	if !strings.Contains(table, "available-high") {
		t.Fatalf("table missing available-high: %q", table)
	}
	if strings.Contains(table, "blocked-task") {
		t.Fatalf("table must not include blocked task: %q", table)
	}
}

func TestFormatTasksJSON(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")

	tasks, err := roadmap.ListAvailableTasks(dir, roadmap.ListTasksOptions{
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("ListAvailableTasks() error = %v", err)
	}

	data, err := roadmap.FormatTasksJSON(tasks)
	if err != nil {
		t.Fatalf("FormatTasksJSON() error = %v", err)
	}

	var items []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Type     string `json:"type"`
		Priority string `json:"priority"`
		Path     string `json:"path"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != "available-high" {
		t.Fatalf("items = %#v, want one available-high entry", items)
	}
}

func taskIDs(tasks []roadmap.Task) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}
