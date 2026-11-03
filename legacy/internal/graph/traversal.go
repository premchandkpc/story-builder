package graph

import "fmt"

func TopologicalSortStrings(nodeIDs []string, edges map[string][]string) ([]string, error) {
	if len(nodeIDs) == 0 {
		return nodeIDs, nil
	}

	inDegree := make(map[string]int, len(nodeIDs))
	for _, id := range nodeIDs {
		inDegree[id] = 0
	}

	for _, tos := range edges {
		for _, to := range tos {
			inDegree[to]++
		}
	}

	queue := make([]string, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	sorted := make([]string, 0, len(nodeIDs))
	visited := make(map[string]bool, len(nodeIDs))

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true
		sorted = append(sorted, id)

		for _, to := range edges[id] {
			inDegree[to]--
			if inDegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}

	if len(sorted) != len(nodeIDs) {
		return nil, fmt.Errorf("graph: cycle detected or %d nodes unreachable (sorted %d)", len(nodeIDs)-len(sorted), len(sorted))
	}

	return sorted, nil
}
