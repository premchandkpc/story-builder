package graph

import (
	"testing"

	"github.com/google/uuid"
)

func newID() uuid.UUID { return uuid.New() }

func TestTopologicalSort_Linear(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{
		{FromNode: a, ToNode: b},
		{FromNode: b, ToNode: c},
	}
	sorted, err := TopologicalSort(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3, got %d", len(sorted))
	}
	pos := map[uuid.UUID]int{}
	for i, n := range sorted {
		pos[n.ID] = i
	}
	if pos[a] > pos[b] || pos[b] > pos[c] {
		t.Fatal("not topologically sorted")
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	a, b, c, d := newID(), newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}, {ID: d}}
	edges := []Edge{
		{FromNode: a, ToNode: b},
		{FromNode: a, ToNode: c},
		{FromNode: b, ToNode: d},
		{FromNode: c, ToNode: d},
	}
	sorted, err := TopologicalSort(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 4 {
		t.Fatalf("expected 4, got %d", len(sorted))
	}
	pos := map[uuid.UUID]int{}
	for i, n := range sorted {
		pos[n.ID] = i
	}
	if pos[a] != 0 {
		t.Fatal("a should be first")
	}
	if pos[d] != 3 {
		t.Fatal("d should be last")
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{
		{FromNode: a, ToNode: b},
		{FromNode: b, ToNode: c},
		{FromNode: c, ToNode: a},
	}
	_, err := TopologicalSort(nodes, edges)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestTopologicalSort_Disconnected(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{
		{FromNode: a, ToNode: b},
	}
	sorted, err := TopologicalSort(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3, got %d", len(sorted))
	}
}

func TestTopologicalSort_Empty(t *testing.T) {
	sorted, err := TopologicalSort(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 0 {
		t.Fatal("expected empty")
	}
}

func TestTopologicalSort_Single(t *testing.T) {
	a := newID()
	sorted, err := TopologicalSort([]Node{{ID: a}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 1 || sorted[0].ID != a {
		t.Fatal("expected single node")
	}
}

func TestPredecessors(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{
		{FromNode: a, ToNode: c},
		{FromNode: b, ToNode: c},
	}
	preds, err := Predecessors(c, nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) != 2 {
		t.Fatalf("expected 2 predecessors, got %d", len(preds))
	}
}

func TestForkJoinEdges(t *testing.T) {
	edges := []Edge{
		{FromNode: newID(), ToNode: newID(), EdgeType: EdgeTypeSeq},
		{FromNode: newID(), ToNode: newID(), EdgeType: EdgeTypeFork},
		{FromNode: newID(), ToNode: newID(), EdgeType: EdgeTypeJoin},
		{FromNode: newID(), ToNode: newID(), EdgeType: EdgeTypeChoice},
		{FromNode: newID(), ToNode: newID(), EdgeType: EdgeTypeParallel},
	}
	result := ForkJoinEdges(edges)
	if len(result) != 2 {
		t.Fatalf("expected 2 fork/join edges, got %d", len(result))
	}
}

func TestIdentifyBranches(t *testing.T) {
	a, b, c, d := newID(), newID(), newID(), newID()
	nodes := []Node{
		{ID: a, Status: NodeStatusAccepted},
		{ID: b, Status: NodeStatusDraft},
		{ID: c, Status: NodeStatusDraft},
		{ID: d, Status: NodeStatusDraft},
	}
	edges := []Edge{
		{FromNode: a, ToNode: b, EdgeType: EdgeTypeFork},
		{FromNode: a, ToNode: c, EdgeType: EdgeTypeFork},
		{FromNode: b, ToNode: d, EdgeType: EdgeTypeSeq},
		{FromNode: c, ToNode: d, EdgeType: EdgeTypeSeq},
	}
	branches, err := IdentifyBranches(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
	if branches[0].ForkNode.ID != a {
		t.Fatal("expected fork node a")
	}
}

func TestBranchCharacterSets(t *testing.T) {
	aID, bID, cID, dID := newID(), newID(), newID(), newID()
	ch1, ch2 := newID(), newID()
	nodes := []Node{
		{ID: aID, CharacterRefs: []uuid.UUID{ch1}},
		{ID: bID, CharacterRefs: []uuid.UUID{ch2}},
		{ID: cID, CharacterRefs: []uuid.UUID{ch1}},
		{ID: dID, CharacterRefs: []uuid.UUID{ch1, ch2}},
	}
	edges := []Edge{
		{FromNode: aID, ToNode: bID, EdgeType: EdgeTypeFork},
		{FromNode: aID, ToNode: cID, EdgeType: EdgeTypeFork},
		{FromNode: bID, ToNode: dID, EdgeType: EdgeTypeSeq},
		{FromNode: cID, ToNode: dID, EdgeType: EdgeTypeSeq},
	}
	sets, err := BranchCharacterSets(nodes, edges, aID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 2 {
		t.Fatalf("expected 2 branch character sets, got %d", len(sets))
	}
}

func TestNodeStatusValid(t *testing.T) {
	if !NodeStatusDraft.Valid() {
		t.Fatal("draft should be valid")
	}
	if !NodeStatusAccepted.Valid() {
		t.Fatal("accepted should be valid")
	}
	if NodeStatus("invalid").Valid() {
		t.Fatal("invalid should not be valid")
	}
}

func TestEdgeTypeBehaviors(t *testing.T) {
	if !EdgeTypeFork.IsBranching() {
		t.Fatal("fork should be branching")
	}
	if EdgeTypeSeq.IsBranching() {
		t.Fatal("seq should not be branching")
	}
	if !EdgeTypeJoin.IsConverging() {
		t.Fatal("join should be converging")
	}
	if EdgeTypeFork.IsConverging() {
		t.Fatal("fork should not be converging")
	}
}
