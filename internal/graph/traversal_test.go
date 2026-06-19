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

// ── TopologicalSortStrings ─────────────────────────────────────────────

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

// ── ValidateDAG ────────────────────────────────────────────────────────

func TestValidateDAG_Valid(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: b, EdgeType: EdgeTypeSeq}, {FromNode: b, ToNode: c, EdgeType: EdgeTypeSeq}}
	if err := ValidateDAG(nodes, edges); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateDAG_DuplicateNode(t *testing.T) {
	id := newID()
	nodes := []Node{{ID: id}, {ID: id}}
	err := ValidateDAG(nodes, nil)
	if err == nil {
		t.Fatal("expected error for duplicate node")
	}
}

func TestValidateDAG_UnknownFromNode(t *testing.T) {
	edges := []Edge{{FromNode: newID(), ToNode: newID()}}
	err := ValidateDAG(nil, edges)
	if err == nil {
		t.Fatal("expected error for unknown from-node")
	}
}

func TestValidateDAG_SelfLoop(t *testing.T) {
	id := newID()
	nodes := []Node{{ID: id}}
	edges := []Edge{{FromNode: id, ToNode: id}}
	err := ValidateDAG(nodes, edges)
	if err == nil {
		t.Fatal("expected error for self-loop")
	}
}

func TestValidateDAG_Cycle(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: b}, {FromNode: b, ToNode: c}, {FromNode: c, ToNode: a}}
	err := ValidateDAG(nodes, edges)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

// ── FindDeadEnds ───────────────────────────────────────────────────────

func TestFindDeadEnds_None(t *testing.T) {
	a, b := newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}}
	edges := []Edge{{FromNode: a, ToNode: b}}
	ends := FindDeadEnds(nodes, edges)
	if len(ends) != 0 {
		t.Fatalf("expected 0 dead ends (last node in topo order excludes ending), got %d", len(ends))
	}
}

func TestFindDeadEnds_Branch(t *testing.T) {
	a, b, c, d := newID(), newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}, {ID: d}}
	edges := []Edge{{FromNode: a, ToNode: b}, {FromNode: a, ToNode: c}, {FromNode: b, ToNode: d}}
	ends := FindDeadEnds(nodes, edges)
	if len(ends) != 1 {
		t.Fatalf("expected 1 dead end (c), got %d (d is last in topo order, not a dead end)", len(ends))
	}
}

func TestFindDeadEnds_Empty(t *testing.T) {
	ends := FindDeadEnds(nil, nil)
	if len(ends) != 0 {
		t.Fatal("expected 0 dead ends")
	}
}

func TestFindDeadEnds_ChainWithBranch(t *testing.T) {
	a, b, c, d, e := newID(), newID(), newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}, {ID: d}, {ID: e}}
	edges := []Edge{
		{FromNode: a, ToNode: b},
		{FromNode: b, ToNode: c},
		{FromNode: b, ToNode: d},
		{FromNode: c, ToNode: e},
	}
	ends := FindDeadEnds(nodes, edges)
	if len(ends) != 1 {
		t.Fatalf("expected 1 dead end (d), got %d", len(ends))
	}
	if ends[0].ID != d {
		t.Fatal("expected dead end d")
	}
}

// ── FindUnreachableScenes ──────────────────────────────────────────────

func TestFindUnreachableScenes_None(t *testing.T) {
	a, b := newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}}
	edges := []Edge{{FromNode: a, ToNode: b}}
	unreach := FindUnreachableScenes(nodes, edges)
	if len(unreach) != 0 {
		t.Fatalf("expected 0 unreachable, got %d", len(unreach))
	}
}

func TestFindUnreachableScenes_Exists(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: b}}
	unreach := FindUnreachableScenes(nodes, edges)
	if len(unreach) != 1 {
		t.Fatalf("expected 1 unreachable (c), got %d", len(unreach))
	}
	if unreach[0].ID != c {
		t.Fatal("expected unreachable node c")
	}
}

func TestFindUnreachableScenes_AllReachable(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: b}, {FromNode: b, ToNode: c}}
	unreach := FindUnreachableScenes(nodes, edges)
	if len(unreach) != 0 {
		t.Fatalf("expected 0 unreachable, got %d", len(unreach))
	}
}

func TestFindUnreachableScenes_Empty(t *testing.T) {
	unreach := FindUnreachableScenes(nil, nil)
	if len(unreach) != 0 {
		t.Fatal("expected 0 unreachable")
	}
}

func TestFindUnreachableScenes_MultipleRoots(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: c}}
	unreach := FindUnreachableScenes(nodes, edges)
	if len(unreach) != 1 {
		t.Fatalf("expected 1 unreachable (b), got %d", len(unreach))
	}
}

// ── FindBranches ───────────────────────────────────────────────────────

func TestFindBranches_Exists(t *testing.T) {
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
	branches, err := FindBranches(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
}

func TestFindBranches_None(t *testing.T) {
	a, b := newID(), newID()
	nodes := []Node{{ID: a, Status: NodeStatusAccepted}, {ID: b, Status: NodeStatusDraft}}
	edges := []Edge{{FromNode: a, ToNode: b, EdgeType: EdgeTypeSeq}}
	branches, err := FindBranches(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected 0 branches, got %d", len(branches))
	}
}

// ── IdentifyBranches edge cases ────────────────────────────────────────

func TestIdentifyBranches_Empty(t *testing.T) {
	branches, err := IdentifyBranches(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 0 {
		t.Fatal("expected 0 branches")
	}
}

func TestIdentifyBranches_NoAcceptedNodes(t *testing.T) {
	a, b := newID(), newID()
	nodes := []Node{{ID: a, Status: NodeStatusDraft}, {ID: b, Status: NodeStatusDraft}}
	edges := []Edge{{FromNode: a, ToNode: b, EdgeType: EdgeTypeFork}}
	branches, err := IdentifyBranches(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected 0 branches (no accepted nodes), got %d", len(branches))
	}
}

func TestIdentifyBranches_ForkNotFromAccepted(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a, Status: NodeStatusAccepted}, {ID: b, Status: NodeStatusDraft}}
	edges := []Edge{{FromNode: b, ToNode: c, EdgeType: EdgeTypeFork}}
	branches, _ := IdentifyBranches(nodes, edges)
	if len(branches) != 0 {
		t.Fatalf("expected 0 branches (fork not from accepted node), got %d", len(branches))
	}
}

// ── Predecessors edge cases ────────────────────────────────────────────

func TestPredecessors_Empty(t *testing.T) {
	preds, err := Predecessors(newID(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) != 0 {
		t.Fatal("expected 0 predecessors")
	}
}

func TestPredecessors_NoEdges(t *testing.T) {
	id := newID()
	preds, err := Predecessors(id, []Node{{ID: id}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) != 0 {
		t.Fatal("expected 0 predecessors")
	}
}

func TestPredecessors_Multiple(t *testing.T) {
	a, b, c := newID(), newID(), newID()
	nodes := []Node{{ID: a}, {ID: b}, {ID: c}}
	edges := []Edge{{FromNode: a, ToNode: c}, {FromNode: b, ToNode: c}}
	preds, err := Predecessors(c, nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) != 2 {
		t.Fatalf("expected 2 predecessors, got %d", len(preds))
	}
}

// ── ForkJoinEdges edge cases ──────────────────────────────────────────

func TestForkJoinEdges_Empty(t *testing.T) {
	result := ForkJoinEdges(nil)
	if len(result) != 0 {
		t.Fatal("expected 0 edges")
	}
}

func TestForkJoinEdges_NoForksJoins(t *testing.T) {
	edges := []Edge{{EdgeType: EdgeTypeSeq}, {EdgeType: EdgeTypeChoice}, {EdgeType: EdgeTypeParallel}}
	result := ForkJoinEdges(edges)
	if len(result) != 0 {
		t.Fatalf("expected 0 fork/join, got %d", len(result))
	}
}

// ── EdgeType.Valid ─────────────────────────────────────────────────────

func TestEdgeTypeValid(t *testing.T) {
	if !EdgeTypeSeq.Valid() {
		t.Fatal("seq should be valid")
	}
	if !EdgeTypeFork.Valid() {
		t.Fatal("fork should be valid")
	}
	if !EdgeTypeJoin.Valid() {
		t.Fatal("join should be valid")
	}
	if !EdgeTypeChoice.Valid() {
		t.Fatal("choice should be valid")
	}
	if !EdgeTypeParallel.Valid() {
		t.Fatal("parallel should be valid")
	}
	if EdgeType("invalid").Valid() {
		t.Fatal("invalid should not be valid")
	}
}

// ── BranchCharacterSets edge cases ─────────────────────────────────────

func TestBranchCharacterSets_NoBranchingEdges(t *testing.T) {
	a := newID()
	nodes := []Node{{ID: a, CharacterRefs: []uuid.UUID{newID()}}}
	sets, err := BranchCharacterSets(nodes, nil, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 0 {
		t.Fatalf("expected 0 char sets (no edges), got %d", len(sets))
	}

	sets, err = BranchCharacterSets(nodes, []Edge{{FromNode: a, ToNode: newID(), EdgeType: EdgeTypeSeq}}, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 0 {
		t.Fatalf("expected 0 char sets (no branching edges), got %d", len(sets))
	}
}

func TestBranchCharacterSets_Deduplicates(t *testing.T) {
	a, b, c, d := newID(), newID(), newID(), newID()
	ch := newID()
	nodes := []Node{
		{ID: a, CharacterRefs: []uuid.UUID{ch}},
		{ID: b, CharacterRefs: []uuid.UUID{ch}},
		{ID: c, CharacterRefs: []uuid.UUID{ch}},
		{ID: d, CharacterRefs: []uuid.UUID{ch}},
	}
	edges := []Edge{
		{FromNode: a, ToNode: b, EdgeType: EdgeTypeFork},
		{FromNode: a, ToNode: c, EdgeType: EdgeTypeFork},
		{FromNode: b, ToNode: d, EdgeType: EdgeTypeSeq},
		{FromNode: c, ToNode: d, EdgeType: EdgeTypeSeq},
	}
	sets, err := BranchCharacterSets(nodes, edges, a)
	if err != nil {
		t.Fatal(err)
	}
	for _, refs := range sets {
		if len(refs) != 1 {
			t.Fatalf("expected 1 unique char ref, got %d", len(refs))
		}
	}
}

// ── ValidateDAG error message ─────────────────────────────────────────

func TestValidationErrorMessage(t *testing.T) {
	err := &ValidationError{Message: "test error"}
	if err.Error() != "test error" {
		t.Fatalf("expected 'test error', got %q", err.Error())
	}
}
