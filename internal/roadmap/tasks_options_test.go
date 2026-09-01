package roadmap

import (
	"strings"
	"testing"
)

func TestParseListTasksOptionsRejectsUnknownSort(t *testing.T) {
	_, err := ParseListTasksOptions(ListTasksOptions{Sort: "title"})
	if err == nil {
		t.Fatal("ParseListTasksOptions() error = nil, want unknown sort error")
	}
	if !strings.Contains(err.Error(), `invalid --sort "title"`) {
		t.Fatalf("error = %q, want invalid --sort message", err.Error())
	}
}

func TestParseListTasksOptionsRejectsUnknownPriority(t *testing.T) {
	_, err := ParseListTasksOptions(ListTasksOptions{Priority: "urgent"})
	if err == nil {
		t.Fatal("ParseListTasksOptions() error = nil, want unknown priority error")
	}
	if !strings.Contains(err.Error(), `invalid --priority "urgent"`) {
		t.Fatalf("error = %q, want invalid --priority message", err.Error())
	}
}

func TestParseListTasksOptionsRejectsUnknownType(t *testing.T) {
	_, err := ParseListTasksOptions(ListTasksOptions{Type: "chore"})
	if err == nil {
		t.Fatal("ParseListTasksOptions() error = nil, want unknown type error")
	}
	if !strings.Contains(err.Error(), `invalid --type "chore"`) {
		t.Fatalf("error = %q, want invalid --type message", err.Error())
	}
}

func TestParseListTasksOptionsDefaultsSortToPriority(t *testing.T) {
	opts, err := ParseListTasksOptions(ListTasksOptions{})
	if err != nil {
		t.Fatalf("ParseListTasksOptions() error = %v", err)
	}
	if opts.Sort != "priority" {
		t.Fatalf("opts.Sort = %q, want %q", opts.Sort, "priority")
	}
}
