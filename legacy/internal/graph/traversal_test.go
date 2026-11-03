package graph

import "testing"

func TestTopologicalSortStrings_Linear(t *testing.T) {
	edges := map[string][]string{"a": {"b"}, "b": {"c"}}
	sorted, err := TopologicalSortStrings([]string{"a", "b", "c"}, edges)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, s := range sorted {
		pos[s] = i
	}
	if pos["a"] > pos["b"] || pos["b"] > pos["c"] {
		t.Fatal("not topologically sorted")
	}
}

func TestTopologicalSortStrings_Cycle(t *testing.T) {
	edges := map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}}
	_, err := TopologicalSortStrings([]string{"a", "b", "c"}, edges)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestTopologicalSortStrings_Empty(t *testing.T) {
	sorted, err := TopologicalSortStrings(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 0 {
		t.Fatal("expected empty")
	}
}

func TestTopologicalSortStrings_Disconnected(t *testing.T) {
	edges := map[string][]string{"a": {"b"}}
	sorted, err := TopologicalSortStrings([]string{"a", "b", "c"}, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 sorted, got %d", len(sorted))
	}
	if sorted[0] != "a" {
		t.Fatal("a should be first")
	}
}

func TestTopologicalSortStrings_Diamond(t *testing.T) {
	edges := map[string][]string{"a": {"b", "c"}, "b": {"d"}, "c": {"d"}}
	sorted, err := TopologicalSortStrings([]string{"a", "b", "c", "d"}, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 4 {
		t.Fatalf("expected 4, got %d", len(sorted))
	}
	pos := map[string]int{}
	for i, s := range sorted {
		pos[s] = i
	}
	if pos["a"] != 0 {
		t.Fatal("a should be first")
	}
	if pos["d"] != 3 {
		t.Fatal("d should be last")
	}
}
