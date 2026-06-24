package graph

import "fmt"

func ValidateDAG(scenes map[string][]string) error {
	if len(scenes) == 0 {
		return nil
	}

	nodeIDs := make([]string, 0, len(scenes))
	for id := range scenes {
		nodeIDs = append(nodeIDs, id)
	}

	return validateAcyclic(nodeIDs, scenes)
}

func validateAcyclic(nodeIDs []string, edges map[string][]string) error {
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

	visited := make(map[string]bool, len(nodeIDs))
	sorted := 0

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true
		sorted++

		for _, to := range edges[id] {
			inDegree[to]--
			if inDegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}

	if sorted != len(nodeIDs) {
		return fmt.Errorf("graph: cycle detected or %d nodes unreachable (sorted %d)", len(nodeIDs)-sorted, sorted)
	}
	return nil
}

func FindDeadEnds(edges map[string][]string) []string {
	outDegree := make(map[string]int)
	for id, tos := range edges {
		if _, ok := outDegree[id]; !ok {
			outDegree[id] = 0
		}
		outDegree[id] += len(tos)
		for _, to := range tos {
			if _, ok := outDegree[to]; !ok {
				outDegree[to] = 0
			}
		}
	}

	var deadEnds []string
	for id, deg := range outDegree {
		if deg == 0 {
			deadEnds = append(deadEnds, id)
		}
	}
	return deadEnds
}

func FindUnreachableScenes(rootSceneID string, edges map[string][]string) []string {
	reachable := make(map[string]bool)
	queue := []string{rootSceneID}
	reachable[rootSceneID] = true

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, to := range edges[id] {
			if !reachable[to] {
				reachable[to] = true
				queue = append(queue, to)
			}
		}
	}

	var unreachable []string
	for id := range edges {
		if !reachable[id] && id != rootSceneID {
			unreachable = append(unreachable, id)
		}
	}
	return unreachable
}

func FindBranches(edges map[string][]string) [][]string {
	outDegree := make(map[string][]string)
	for id, tos := range edges {
		if len(tos) > 1 {
			outDegree[id] = tos
		}
	}

	var branches [][]string
	for _, tos := range outDegree {
		if len(tos) > 1 {
			branches = append(branches, tos)
		}
	}
	return branches
}

func FindMergePoints(edges map[string][]string) []string {
	inDegree := make(map[string]int)
	for _, tos := range edges {
		for _, to := range tos {
			inDegree[to]++
		}
	}

	var merges []string
	for id, deg := range inDegree {
		if deg > 1 {
			merges = append(merges, id)
		}
	}
	return merges
}
