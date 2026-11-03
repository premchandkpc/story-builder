//go:build test

package graph

import (
	"math/rand"
	"testing"
)

func generateDAGStrings(numNodes int, edgeProb float64) ([]string, map[string][]string) {
	ids := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		ids[i] = id(i)
	}
	edges := make(map[string][]string)
	for i := 0; i < numNodes; i++ {
		for j := i + 1; j < numNodes; j++ {
			if rand.Float64() < edgeProb {
				edges[ids[i]] = append(edges[ids[i]], ids[j])
			}
		}
	}
	return ids, edges
}

func generateGraphWithCycle(numNodes int) ([]string, map[string][]string) {
	ids := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		ids[i] = id(i)
	}
	edges := make(map[string][]string)
	for i := 0; i < numNodes-1; i++ {
		edges[ids[i]] = append(edges[ids[i]], ids[i+1])
	}
	edges[ids[numNodes-1]] = append(edges[ids[numNodes-1]], ids[0])
	return ids, edges
}

func id(n int) string {
	if n < 26 {
		return string(rune('A' + n))
	}
	return string(rune('A'+n%26)) + string(rune('a'+(n/26)%26))
}

func TestGraph_TopologicalSort_Acyclic(t *testing.T) {
	for n := 0; n < 100; n++ {
		ids, edges := generateDAGStrings(15, 0.25)
		sorted, err := TopologicalSortStrings(ids, edges)
		if err != nil {
			t.Fatalf("iteration %d: expected acyclic DAG, got: %v", n, err)
		}
		if len(sorted) != len(ids) {
			t.Errorf("iteration %d: sorted %d nodes, want %d", n, len(sorted), len(ids))
		}
	}
}

func TestGraph_TopologicalSort_DetectsCycle(t *testing.T) {
	ids, edges := generateGraphWithCycle(5)
	_, err := TopologicalSortStrings(ids, edges)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestGraph_TopologicalSort_AllNodesInOrder(t *testing.T) {
	for n := 0; n < 100; n++ {
		ids, edges := generateDAGStrings(10, 0.3)
		sorted, err := TopologicalSortStrings(ids, edges)
		if err != nil {
			t.Fatalf("iteration %d: sort failed: %v", n, err)
		}
		seen := make(map[string]int)
		for i, id := range sorted {
			seen[id] = i
		}
		for from, tos := range edges {
			for _, to := range tos {
				if seen[from] > seen[to] {
					t.Errorf("iteration %d: edge %s->%s violates order", n, from, to)
				}
			}
		}
	}
}

func TestGraph_TopologicalSort_SingleNode(t *testing.T) {
	sorted, err := TopologicalSortStrings([]string{"A"}, nil)
	if err != nil {
		t.Fatalf("single node: %v", err)
	}
	if len(sorted) != 1 || sorted[0] != "A" {
		t.Fatalf("single node: got %v", sorted)
	}
}

func TestGraph_TopologicalSort_Empty(t *testing.T) {
	sorted, err := TopologicalSortStrings([]string{}, nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if len(sorted) != 0 {
		t.Fatalf("empty: got %d", len(sorted))
	}
}

func TestGraph_TopologicalSort_Disconnected(t *testing.T) {
	ids := []string{"A", "B", "C"}
	edges := map[string][]string{}
	sorted, err := TopologicalSortStrings(ids, edges)
	if err != nil {
		t.Fatalf("disconnected: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("disconnected: got %d", len(sorted))
	}
}
