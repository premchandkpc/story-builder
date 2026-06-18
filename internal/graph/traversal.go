package graph

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// TopologicalSortStrings is a Kahn's algorithm variant for string-keyed graphs.
// edges maps each node ID to its outgoing neighbor IDs.
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

func TopologicalSort(nodes []Node, edges []Edge) ([]Node, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	inDegree := make(map[uuid.UUID]int)
	nodeMap := make(map[uuid.UUID]*Node)
	for i := range nodes {
		inDegree[nodes[i].ID] = 0
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, e := range edges {
		adj[e.FromNode] = append(adj[e.FromNode], e.ToNode)
		inDegree[e.ToNode]++
	}

	var queue []uuid.UUID
	for _, n := range nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var sorted []Node
	visited := make(map[uuid.UUID]bool)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true

		if n, ok := nodeMap[id]; ok {
			sorted = append(sorted, *n)
		}

		for _, neighbor := range adj[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(sorted) != len(nodes) {
		return nil, fmt.Errorf("graph: cycle detected or disconnected nodes")
	}

	return sorted, nil
}

func Predecessors(nodeID uuid.UUID, nodes []Node, edges []Edge) ([]Node, error) {
	nodeMap := make(map[uuid.UUID]Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var preds []Node
	for _, e := range edges {
		if e.ToNode == nodeID {
			if n, ok := nodeMap[e.FromNode]; ok {
				preds = append(preds, n)
			}
		}
	}
	return preds, nil
}

func ForkJoinEdges(edges []Edge) []Edge {
	var result []Edge
	for _, e := range edges {
		if e.EdgeType == EdgeTypeFork || e.EdgeType == EdgeTypeJoin {
			result = append(result, e)
		}
	}
	return result
}

type Branch struct {
	ForkNode   Node
	BranchPath []Node
	EndNode    *Node
}

func IdentifyBranches(nodes []Node, edges []Edge) ([]Branch, error) {
	nodeMap := make(map[uuid.UUID]Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	adj := make(map[uuid.UUID][]Edge)
	for _, e := range edges {
		adj[e.FromNode] = append(adj[e.FromNode], e)
	}

	var branches []Branch
	for _, n := range nodes {
		if n.Status != NodeStatusAccepted {
			continue
		}

		outEdges := adj[n.ID]
		var forkEdges []Edge
		for _, e := range outEdges {
			if e.EdgeType == EdgeTypeFork || e.EdgeType == EdgeTypeChoice {
				forkEdges = append(forkEdges, e)
			}
		}

		if len(forkEdges) >= 2 {
			b := Branch{ForkNode: n}
			for _, fe := range forkEdges {
				path := walkToJoin(fe.ToNode, adj, nodeMap)
				b.BranchPath = append(b.BranchPath, path...)
			}
			branches = append(branches, b)
		}
	}
	return branches, nil
}

func walkToJoin(start uuid.UUID, adj map[uuid.UUID][]Edge, nodeMap map[uuid.UUID]Node) []Node {
	visited := make(map[uuid.UUID]bool)
	var path []Node
	queue := []uuid.UUID{start}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true

		if n, ok := nodeMap[id]; ok {
			path = append(path, n)
		}

		for _, e := range adj[id] {
			if e.EdgeType == EdgeTypeJoin {
				if n, ok := nodeMap[e.ToNode]; ok {
					path = append(path, n)
				}
				return path
			}
			if !visited[e.ToNode] {
				queue = append(queue, e.ToNode)
			}
		}
	}
	return path
}

func BranchCharacterSets(nodes []Node, edges []Edge, forkNodeID uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	adj := make(map[uuid.UUID][]Edge)
	for _, e := range edges {
		adj[e.FromNode] = append(adj[e.FromNode], e)
	}

	nodeMap := make(map[uuid.UUID]Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	forkEdges := adj[forkNodeID]
	result := make(map[uuid.UUID][]uuid.UUID)

	for _, fe := range forkEdges {
		if !fe.EdgeType.IsBranching() {
			continue
		}
		pathNodes := walkToJoin(fe.ToNode, adj, nodeMap)
		var refs []uuid.UUID
		for _, pn := range pathNodes {
			refs = append(refs, pn.CharacterRefs...)
		}
		dedup := make(map[uuid.UUID]bool)
		var unique []uuid.UUID
		for _, r := range refs {
			if !dedup[r] {
				dedup[r] = true
				unique = append(unique, r)
			}
		}
		sort.Slice(unique, func(i, j int) bool {
			return unique[i].String() < unique[j].String()
		})
		result[fe.ToNode] = unique
	}
	return result, nil
}
