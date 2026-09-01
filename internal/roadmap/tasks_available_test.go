package roadmap

import "testing"

func TestFilterAvailableBlocksDependencyCycle(t *testing.T) {
	tasks := []Task{
		{ID: "task-a", DependsOn: []string{"task-b"}},
		{ID: "task-b", DependsOn: []string{"task-a"}},
	}

	available := filterAvailable(tasks)
	if len(available) != 0 {
		t.Fatalf("filterAvailable() = %#v, want no available tasks for dependency cycle", available)
	}
}
