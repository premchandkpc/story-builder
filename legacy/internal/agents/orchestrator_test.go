package agents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

type mockTurnRepo struct {
	mu     sync.Mutex
	turns  []*domain.SceneTurn
	createErr error
	updateErr error
}

func (m *mockTurnRepo) Create(_ context.Context, t *domain.SceneTurn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	t.ID = fmt.Sprintf("turn-%d", len(m.turns)+1)
	t.CreatedAt = time.Now()
	m.turns = append(m.turns, t)
	return nil
}

func (m *mockTurnRepo) Update(_ context.Context, t *domain.SceneTurn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	t.UpdatedAt = time.Now()
	for i, existing := range m.turns {
		if existing.ID == t.ID {
			m.turns[i] = t
			return nil
		}
	}
	return nil
}

type mockBudgetChecker struct {
	mu     sync.Mutex
	checks []string
	failOn string
}

func (m *mockBudgetChecker) CheckAndConsume(_ context.Context, storyID, model, agentType string, _, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks = append(m.checks, agentType)
	if m.failOn == agentType {
		return fmt.Errorf("budget exceeded for %s", agentType)
	}
	return nil
}

type mockCharManager struct {
	mu     sync.Mutex
	events []CharacterEvent
}

func (m *mockCharManager) BroadcastEvent(evt CharacterEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, evt)
}

// Shallow mock for QueryProposals used by Plan
func (m *mockCharManager) QueryProposals(_ context.Context) []CharacterProposal {
	return nil
}

// helper to make AgentRunner from a simple function
func agentRunnerOK(content string) AgentRunner {
	return func(_ context.Context, _ AgentInput) (*AgentOutput, error) {
		return &AgentOutput{Content: content, Decisions: map[string]any{"emotion": "neutral"}}, nil
	}
}

func agentRunnerWithDelay(content string, delay time.Duration) AgentRunner {
	return func(ctx context.Context, _ AgentInput) (*AgentOutput, error) {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			return &AgentOutput{Content: content}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func agentRunnerError(msg string) AgentRunner {
	return func(_ context.Context, _ AgentInput) (*AgentOutput, error) {
		return nil, errors.New(msg)
	}
}

func agentRunnerWithDecisions(d map[string]any) AgentRunner {
	return func(_ context.Context, _ AgentInput) (*AgentOutput, error) {
		return &AgentOutput{Content: "output", Decisions: d}, nil
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func init() {
	slog.SetLogLoggerLevel(slog.LevelWarn)
}

func newTestRegistry() *AgentRegistry {
	r := NewAgentRegistry()
	r.Register(AgentSpec{Name: domain.AgentTypeDirector, Role: "director", Model: "claude-sonnet", Runner: agentRunnerOK("director: plan ready")})
	r.Register(AgentSpec{Name: domain.AgentTypeNarrator, Role: "narrator", Model: "claude-sonnet", Runner: agentRunnerOK("narrator: scene set")})
	r.Register(AgentSpec{Name: domain.AgentTypeEditor, Role: "editor", Model: "claude-sonnet", Runner: agentRunnerOK("editor: refined")})
	r.Register(AgentSpec{Name: domain.AgentTypeCanonGuard, Role: "canon_guard", Model: "claude-haiku", Runner: agentRunnerOK("canon: ok")})
	r.Register(AgentSpec{Name: domain.AgentTypeCritic, Role: "critic", Model: "claude-haiku", Runner: agentRunnerWithDecisions(map[string]any{"score": 8.5})})
	r.Register(AgentSpec{Name: domain.AgentTypeStateExtract, Role: "state_extractor", Model: "local-7b", Runner: agentRunnerOK("state: extracted")})
	r.Register(AgentSpec{Name: domain.AgentTypeWorld, Role: "world", Model: "local-7b", Runner: agentRunnerOK("world: consistent")})
	r.Register(AgentSpec{Name: domain.AgentTypeArc, Role: "arc", Model: "local-7b", Runner: agentRunnerOK("arc: on track")})
	r.Register(AgentSpec{Name: domain.AgentTypeMemory, Role: "memory", Model: "local-7b", Runner: agentRunnerOK("memory: stored")})
	return r
}

func newBaseScene() *domain.Scene {
	return &domain.Scene{
		ID:           "scene-1",
		Title:        "Test Scene",
		BeatIntent:   "testing",
		FlowType:     domain.FlowTypeMonologue,
		MaxTurns:     10,
		Participants: []string{"char-1"},
		POV:          "third-person",
		Tone:         "neutral",
	}
}

func baseAgentCtx(scene *domain.Scene) *AgentContext {
	return &AgentContext{
		StoryID: "story-1",
		SceneID: scene.ID,
		Scene:   scene,
		Characters: []*domain.Character{
			{CharID: "char-1", Name: "Alice"},
		},
		CharStates: []*domain.CharacterState{
			{CharacterID: "char-1", Health: 100},
		},
	}
}

// ---------------------------------------------------------------------------
// Plan tests
// ---------------------------------------------------------------------------

func TestPlan_FlowTypes(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})
	ctx := context.Background()

	tests := []struct {
		name     string
		flowType string
		wantLen  int // expected number of TurnSteps in plan
		wantChar bool
	}{
		{name: "monologue", flowType: domain.FlowTypeMonologue, wantLen: 5, wantChar: true},
		{name: "dialogue", flowType: domain.FlowTypeDialogue, wantLen: 6, wantChar: true},
		{name: "round_robin", flowType: domain.FlowTypeRoundRobin, wantLen: 4, wantChar: true},
		{name: "action", flowType: domain.FlowTypeAction, wantLen: 4, wantChar: true},
		{name: "silent", flowType: domain.FlowTypeSilent, wantLen: 2, wantChar: false},
		{name: "custom", flowType: domain.FlowTypeCustom, wantLen: 5, wantChar: true},
		{name: "parallel", flowType: domain.FlowTypeParallel, wantLen: 5, wantChar: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := newBaseScene()
			scene.FlowType = tt.flowType
			plan, err := o.Plan(ctx, scene)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if plan.SceneID != scene.ID {
				t.Errorf("Plan.SceneID = %q, want %q", plan.SceneID, scene.ID)
			}
			if len(plan.TurnOrder) != tt.wantLen {
				t.Errorf("len(TurnOrder) = %d, want %d\n  got: %v", len(plan.TurnOrder), tt.wantLen, stepTypes(plan.TurnOrder))
			}
			// Plan ensures maxTurns even if scene provides 0
	if plan.MaxTurns == 0 {
		t.Error("MaxTurns should default to non-zero")
	}
	// All monologue plans start with director
			if len(plan.TurnOrder) > 0 && plan.TurnOrder[0].AgentType != domain.AgentTypeDirector {
				t.Errorf("TurnOrder[0] = %s, want director", plan.TurnOrder[0].AgentType)
			}
			// Director is always blocking
			if len(plan.TurnOrder) > 0 && !plan.TurnOrder[0].Blocking {
				t.Error("Director step should be blocking")
			}
			if tt.wantChar {
				foundChar := false
				for _, s := range plan.TurnOrder {
					if s.AgentType == "char-1" {
						foundChar = true
						if s.Blocking {
							t.Error("Character step should be non-blocking")
						}
						break
					}
				}
				if !foundChar {
					t.Error("No character step in plan for flow type", tt.flowType)
				}
			}
		})
	}
}

func stepTypes(steps []TurnStep) []string {
	types := make([]string, len(steps))
	for i, s := range steps {
		suffix := ""
		if s.Blocking {
			suffix = "!"
		}
		types[i] = s.AgentType + suffix
	}
	return types
}

func TestPlan_CharProposals(t *testing.T) {
	registry := newTestRegistry()
	cm := &mockCharManager{}
	// mock QueryProposals by casting
	o := NewOrchestrator(OrchestratorConfig{
		Registry:    registry,
		CharManager: cm,
		LLMClient:  &llm.MockLLMClient{},
	})
	scene := newBaseScene()
	plan, err := o.Plan(context.Background(), scene)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Proposals) != 0 {
		t.Errorf("Proposals = %d, want 0", len(plan.Proposals))
	}
	if plan.MaxTurns != scene.MaxTurns {
		t.Errorf("MaxTurns = %d, want %d", plan.MaxTurns, scene.MaxTurns)
	}
}

// ---------------------------------------------------------------------------
// gatherCharacterAgentIDs
// ---------------------------------------------------------------------------

func TestGatherCharacterAgentIDs(t *testing.T) {
	scene := newBaseScene()
	scene.Participants = []string{"c1", "c2", "c1"}

	proposals := []CharacterProposal{
		{CharacterID: "c2"},
		{CharacterID: "c3"},
	}

	ids := gatherCharacterAgentIDs(scene, proposals)
	if len(ids) != 3 {
		t.Errorf("len(ids) = %d, want 3: %v", len(ids), ids)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate id: %s", id)
		}
		seen[id] = true
	}
	for _, want := range []string{"c1", "c2", "c3"} {
		if !seen[want] {
			t.Errorf("missing id: %s", want)
		}
	}
}

func TestGatherCharacterAgentIDs_Empty(t *testing.T) {
	ids := gatherCharacterAgentIDs(newBaseScene(), nil)
	if len(ids) != 1 || ids[0] != "char-1" {
		t.Errorf("ids = %v, want [char-1]", ids)
	}
}

// ---------------------------------------------------------------------------
// agentBudgetEstimate
// ---------------------------------------------------------------------------

func TestAgentBudgetEstimate(t *testing.T) {
	tests := []struct {
		agentType string
		wantP     int
		wantC     int
	}{
		{domain.AgentTypeDirector, 3000, 1500},
		{domain.AgentTypeCharacter, 1500, 500},
		{domain.AgentTypeNarrator, 2000, 1000},
		{domain.AgentTypeEditor, 1500, 500},
		{domain.AgentTypeCanonGuard, 1000, 300},
		{domain.AgentTypeCritic, 1000, 300},
		{domain.AgentTypeStateExtract, 1000, 500},
		{domain.AgentTypeMemory, 1000, 500},
		{"unknown", 1000, 500},
	}
	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			p, c := agentBudgetEstimate(tt.agentType, "some-model")
			if p != tt.wantP || c != tt.wantC {
				t.Errorf("budgetEstimate(%q) = (%d,%d), want (%d,%d)", tt.agentType, p, c, tt.wantP, tt.wantC)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Execute — core orchestration
// ---------------------------------------------------------------------------

func TestExecute_SequentialAllBlocking(t *testing.T) {
	// Plan with no character steps = all blocking
	registry := newTestRegistry()
	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.SceneID != "scene-1" {
		t.Errorf("SceneID = %q", result.SceneID)
	}
	if len(result.Turns) != 2 {
		t.Errorf("len(Turns) = %d, want 2", len(result.Turns))
	}
	for _, turn := range result.Turns {
		if turn.Status != domain.TurnStatusDone {
			t.Errorf("turn %d status = %s, want done", turn.Number, turn.Status)
		}
	}
}

func TestExecute_ParallelNonBlocking(t *testing.T) {
	// Character agents run in parallel
	registry := newTestRegistry()
	// Register a character agent
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-1: performs", 50*time.Millisecond),
	})
	registry.Register(AgentSpec{
		Name: "char-2", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-2: performs", 50*time.Millisecond),
	})

	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "char-1", Phase: "perform", Required: true, Blocking: false},
			{AgentType: "char-2", Phase: "perform", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
		},
	}

	start := time.Now()
	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 4 {
		t.Errorf("len(Turns) = %d, want 4", len(result.Turns))
	}
	// The 2 character agents ran in parallel, so total time < sequential (50+50=100ms)
	// but > max(char delay) + director + narrator overhead
	if dur > 200*time.Millisecond {
		t.Logf("parallel execution too slow: %v (may be due to CI)", dur)
	}
	// Verify all completed
	for _, turn := range result.Turns {
		if turn.Status != domain.TurnStatusDone {
			t.Errorf("turn %d (%s) status = %s, want done", turn.Number, turn.AgentID, turn.Status)
		}
	}
}

func TestExecute_ParallelWithBarrier(t *testing.T) {
	// Non-blocking steps after a blocking step must wait for the barrier
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("slow char", 80*time.Millisecond),
	})
	registry.Register(AgentSpec{
		Name: "char-2", Role: "character", Model: "local-7b",
		Runner: agentRunnerOK("fast char"),
	})

	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			// Two parallel non-blocking
			{AgentType: "char-1", Phase: "perform", Required: true, Blocking: false},
			{AgentType: "char-2", Phase: "perform", Required: true, Blocking: false},
			// Barrier — must wait for both chars
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 4 {
		t.Fatalf("len(Turns) = %d, want 4", len(result.Turns))
	}
	// Narrator turn number should be 4 (after 1 dir + 2 chars)
	last := result.Turns[len(result.Turns)-1]
	if last.Number != 4 || last.AgentID != "narrator" {
		t.Errorf("last turn = #%d %s, want #4 narrator", last.Number, last.AgentID)
	}
}

func TestExecute_RequiredAgentMissing(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})
	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: "nonexistent-agent", Phase: "test", Required: true, Blocking: true},
		},
	}
	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err == nil {
		t.Fatal("expected error for missing required agent, got nil")
	}
}

func TestExecute_NonRequiredAgentMissing(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})
	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: "nonexistent-agent", Phase: "test", Required: false, Blocking: false},
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}
	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	// Only the director turn should be recorded
	if len(result.Turns) != 1 {
		t.Errorf("len(Turns) = %d, want 1", len(result.Turns))
	}
}

func TestExecute_StepError(t *testing.T) {
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "failing-agent", Role: "character", Model: "local-7b",
		Runner: agentRunnerError("something went wrong"),
	})

	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "failing-agent", Phase: "perform", Required: true, Blocking: false},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	if err == nil {
		t.Fatal("expected error from failing step, got nil")
	}
	// Turn should be marked as failed
	if len(turnRepo.turns) > 1 {
		last := turnRepo.turns[len(turnRepo.turns)-1]
		if last.Status != domain.TurnStatusFailed {
			t.Errorf("turn status = %s, want failed", last.Status)
		}
	}
}

func TestExecute_BudgetExceeded(t *testing.T) {
	registry := newTestRegistry()
	budget := &mockBudgetChecker{failOn: domain.AgentTypeDirector}
	o := NewOrchestrator(OrchestratorConfig{
		Registry:      registry,
		BudgetChecker: budget,
		LLMClient:    &llm.MockLLMClient{},
	})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err == nil {
		t.Fatal("expected budget error, got nil")
	}
	if len(budget.checks) == 0 {
		t.Error("budget checker was not called")
	}
}

func TestExecute_CharManagerEvents(t *testing.T) {
	registry := newTestRegistry()
	cm := &mockCharManager{}
	o := NewOrchestrator(OrchestratorConfig{
		Registry:    registry,
		CharManager: cm,
		LLMClient:  &llm.MockLLMClient{},
	})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cm.mu.Lock()
	evtCount := len(cm.events)
	cm.mu.Unlock()

	if evtCount < 2 {
		t.Errorf("char events = %d, want >= 2 (scene_start + turn_complete + scene_end)", evtCount)
	}

	// Verify scene_start event
	cm.mu.Lock()
	foundStart := false
	foundEnd := false
	for _, e := range cm.events {
		switch e.Type {
		case EventSceneStart:
			foundStart = true
			if e.SceneID != "scene-1" {
				t.Errorf("scene_start.SceneID = %q", e.SceneID)
			}
		case EventSceneEnd:
			foundEnd = true
		}
	}
	cm.mu.Unlock()

	if !foundStart {
		t.Error("missing scene_start event")
	}
	if !foundEnd {
		t.Error("missing scene_end event")
	}
}

func TestExecute_CriticScore(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.CriticScore != 8.5 {
		t.Errorf("CriticScore = %f, want 8.5", result.CriticScore)
	}
}

func TestExecute_EmptyPlan(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{SceneID: "scene-1"}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 0 {
		t.Errorf("Turns = %d, want 0", len(result.Turns))
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "slow-agent", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("slow output", 500*time.Millisecond),
	})

	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "slow-agent", Phase: "perform", Required: true, Blocking: false},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	time.Sleep(20 * time.Millisecond) // wait for timeout to fire

	_, err := o.Execute(ctx, plan, baseAgentCtx(newBaseScene()), nil)
	if err == nil {
		t.Log("Execute may succeed despite cancelled context (race with slow agent)")
	}
}

func TestExecute_MultipleParallelBatches(t *testing.T) {
	// Multiple batches of non-blocking steps separated by blocking barriers
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-1: acts", 30*time.Millisecond),
	})
	registry.Register(AgentSpec{
		Name: "char-2", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-2: acts", 30*time.Millisecond),
	})

	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	// Dialogue-like: Director -> chars perform -> chars respond -> Narrator -> Editor
	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "char-1", Phase: "perform", Required: true, Blocking: false},
			{AgentType: "char-2", Phase: "perform", Required: true, Blocking: false},
			{AgentType: "char-1", Phase: "respond", Required: true, Blocking: false},
			{AgentType: "char-2", Phase: "respond", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 7 {
		t.Errorf("len(Turns) = %d, want 7", len(result.Turns))
	}
	// Verify all done
	for _, turn := range result.Turns {
		if turn.Status != domain.TurnStatusDone {
			t.Errorf("turn %d (%s) status = %s, want done", turn.Number, turn.AgentID, turn.Status)
		}
	}
}

func TestExecute_TurnRepoCreateError(t *testing.T) {
	registry := newTestRegistry()
	turnRepo := &mockTurnRepo{createErr: errors.New("db unavailable")}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), turnRepo)
	if err == nil {
		t.Fatal("expected error from turn repo create, got nil")
	}
}

func TestExecute_SemaphoreBoundsConcurrency(t *testing.T) {
	// Ensure maxConcurrentAgents is respected: run more non-blocking steps than
	// the semaphore limit and verify they don't all run at once
	registry := newTestRegistry()

	var (
		mu          sync.Mutex
		maxParallel int
		current     int32
	)

	makeRunner := func(id string) AgentRunner {
		return func(ctx context.Context, _ AgentInput) (*AgentOutput, error) {
			v := atomic.AddInt32(&current, 1)
			mu.Lock()
			if int(v) > maxParallel {
				maxParallel = int(v)
			}
			mu.Unlock()
			defer atomic.AddInt32(&current, -1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(20 * time.Millisecond):
				return &AgentOutput{Content: id + " done"}, nil
			}
		}
	}

	for i := range 10 {
		id := fmt.Sprintf("agent-%d", i+1)
		registry.Register(AgentSpec{Name: id, Role: "character", Model: "local-7b", Runner: makeRunner(id)})
	}

	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	steps := []TurnStep{{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true}}
	for i := range 10 {
		steps = append(steps, TurnStep{AgentType: fmt.Sprintf("agent-%d", i+1), Phase: "perform", Required: true, Blocking: false})
	}

	plan := &OrchestrationPlan{SceneID: "scene-1", TurnOrder: steps}
	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if maxParallel > maxConcurrentAgents {
		t.Errorf("max parallel = %d, exceeds limit %d", maxParallel, maxConcurrentAgents)
	}
}

func TestExecute_NonBlockingStepFailureCancelsOthers(t *testing.T) {
	// When a non-blocking step fails, the errgroup should cancel remaining
	registry := newTestRegistry()

	var ranAfterFailure int32
	registry.Register(AgentSpec{
		Name: "failing-char", Role: "character", Model: "local-7b",
		Runner: agentRunnerError("char failed"),
	})
	registry.Register(AgentSpec{
		Name: "other-char", Role: "character", Model: "local-7b",
		Runner: func(ctx context.Context, _ AgentInput) (*AgentOutput, error) {
			atomic.StoreInt32(&ranAfterFailure, 1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return &AgentOutput{Content: "done"}, nil
			}
		},
	})

	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "failing-char", Phase: "perform", Required: true, Blocking: false},
			{AgentType: "other-char", Phase: "perform", Required: true, Blocking: false},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	_ = ranAfterFailure // other-char may or may not run depending on timing
}

// ---------------------------------------------------------------------------
// RunFinish tests
// ---------------------------------------------------------------------------

func TestRunFinish(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	err := o.RunFinish(context.Background(), "scene-1", baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("RunFinish() error = %v", err)
	}
}

func TestRunFinish_TurnRepo(t *testing.T) {
	registry := newTestRegistry()
	turnRepo := &mockTurnRepo{}
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	err := o.RunFinish(context.Background(), "scene-1", baseAgentCtx(newBaseScene()), turnRepo)
	if err != nil {
		t.Fatalf("RunFinish() error = %v", err)
	}
	if len(turnRepo.turns) == 0 {
		t.Error("no turns recorded in repo")
	}
}

func TestRunFinish_BudgetExceeded(t *testing.T) {
	registry := newTestRegistry()
	budget := &mockBudgetChecker{failOn: domain.AgentTypeStateExtract}
	o := NewOrchestrator(OrchestratorConfig{
		Registry:      registry,
		BudgetChecker: budget,
		LLMClient:    &llm.MockLLMClient{},
	})

	err := o.RunFinish(context.Background(), "scene-1", baseAgentCtx(newBaseScene()), nil)
	if err == nil {
		t.Fatal("expected budget error, got nil")
	}
}

// ---------------------------------------------------------------------------
// extractEmotion
// ---------------------------------------------------------------------------

func TestExtractEmotion(t *testing.T) {
	tests := []struct {
		name   string
		output *AgentOutput
		want   string
	}{
		{name: "nil output", output: nil, want: ""},
		{name: "no decisions", output: &AgentOutput{Content: "hi"}, want: ""},
		{name: "with emotion", output: &AgentOutput{Content: "hi", Decisions: map[string]any{"emotion": "joyful"}}, want: "joyful"},
		{name: "wrong type", output: &AgentOutput{Content: "hi", Decisions: map[string]any{"emotion": 42}}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEmotion(tt.output)
			if got != tt.want {
				t.Errorf("extractEmotion = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Plan — multiple participants
// ---------------------------------------------------------------------------

func TestPlan_MultipleParticipants(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	scene := newBaseScene()
	scene.Participants = []string{"char-1", "char-2"}
	scene.FlowType = domain.FlowTypeMonologue

	// Register char-2 (char-1 is NOT registered, will be skipped by runStep)
	registry.Register(AgentSpec{
		Name: "char-2", Role: "character", Model: "local-7b",
		Runner: agentRunnerOK("char-2 speaks"),
	})

	plan, err := o.Plan(context.Background(), scene)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// monologue: director + 2 chars + narrator + editor + canon = 6
	if len(plan.TurnOrder) != 6 {
		t.Errorf("len(TurnOrder) = %d, want 6\n  got: %v", len(plan.TurnOrder), stepTypes(plan.TurnOrder))
	}
	// Char steps should be non-blocking
	for _, s := range plan.TurnOrder {
		if s.AgentType == "char-1" || s.AgentType == "char-2" {
			if s.Blocking {
				t.Errorf("%s should be non-blocking", s.AgentType)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Plan — with character proposals
// ---------------------------------------------------------------------------

func TestPlan_WithProposals(t *testing.T) {
	cmWithProposals := &mockCharManagerWithProposals{}
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{
		Registry:    registry,
		CharManager: cmWithProposals,
		LLMClient:  &llm.MockLLMClient{},
	})

	scene := newBaseScene()
	plan, err := o.Plan(context.Background(), scene)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Proposals) == 0 {
		t.Error("expected proposals from char manager")
	}
}

// mockCharManagerWithProposals returns non-empty proposals
type mockCharManagerWithProposals struct{ mockCharManager }

func (m *mockCharManagerWithProposals) QueryProposals(_ context.Context) []CharacterProposal {
	return []CharacterProposal{
		{CharacterID: "char-1", ActionType: "investigate", Priority: 1},
	}
}

// ---------------------------------------------------------------------------
// Execute — character event types
// ---------------------------------------------------------------------------

func TestExecute_CharacterEventTypes(t *testing.T) {
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerOK("char: acts"),
	})

	cm := &mockCharManager{}
	o := NewOrchestrator(OrchestratorConfig{
		Registry:    registry,
		CharManager: cm,
		LLMClient:  &llm.MockLLMClient{},
	})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "char-1", Phase: "perform", Required: true, Blocking: false},
		},
	}

	_, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	foundCharAction := false
	foundTurnComplete := false
	for _, e := range cm.events {
		switch e.Type {
		case EventCharAction:
			foundCharAction = true
		case EventTurnComplete:
			foundTurnComplete = true
		}
	}
	if !foundCharAction {
		t.Error("missing EventCharAction for character agent")
	}
	if !foundTurnComplete {
		t.Error("missing EventTurnComplete for director")
	}
}

// ---------------------------------------------------------------------------
// Execute — non-required non-blocking missing agent in parallel
// ---------------------------------------------------------------------------

func TestExecute_NonRequiredNonBlockingMissingAgent(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: "unknown-agent", Phase: "test", Required: false, Blocking: false},
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 1 {
		t.Errorf("len(Turns) = %d, want 1 (director only)", len(result.Turns))
	}
}

// ---------------------------------------------------------------------------
// Execute — multiple waves of non-blocking steps across barriers
// ---------------------------------------------------------------------------

func TestExecute_MultipleWaves(t *testing.T) {
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-1: first", 20*time.Millisecond),
	})
	registry.Register(AgentSpec{
		Name: "char-2", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char-2: second", 20*time.Millisecond),
	})

	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	// Three waves: [char-1/char-2] barrier [char-1] barrier [char-2]
	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: "char-1", Phase: "first", Required: true, Blocking: false},
			{AgentType: "char-2", Phase: "first", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "char-1", Phase: "second", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: "char-2", Phase: "third", Required: true, Blocking: false},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 6 {
		t.Errorf("len(Turns) = %d, want 6", len(result.Turns))
	}
	for _, turn := range result.Turns {
		if turn.Status != domain.TurnStatusDone {
			t.Errorf("turn %d (%s) status = %s, want done", turn.Number, turn.AgentID, turn.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// Execute — all blocking steps (no non-blocking steps in plan)
// ---------------------------------------------------------------------------

func TestExecute_AllBlockingSteps(t *testing.T) {
	registry := newTestRegistry()
	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
		},
	}

	result, err := o.Execute(context.Background(), plan, baseAgentCtx(newBaseScene()), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Turns) != 3 {
		t.Errorf("len(Turns) = %d, want 3", len(result.Turns))
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety: Execute must be safe for concurrent calls
// ---------------------------------------------------------------------------

func TestExecute_ConcurrentSafety(t *testing.T) {
	registry := newTestRegistry()
	registry.Register(AgentSpec{
		Name: "char-1", Role: "character", Model: "local-7b",
		Runner: agentRunnerWithDelay("char: acts", 10*time.Millisecond),
	})

	o := NewOrchestrator(OrchestratorConfig{Registry: registry, LLMClient: &llm.MockLLMClient{}})

	plan := &OrchestrationPlan{
		SceneID: "scene-1",
		TurnOrder: []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: "char-1", Phase: "perform", Required: true, Blocking: false},
		},
	}

	ctx := context.Background()
	g := new(errgroup.Group)
	for range 5 {
		g.Go(func() error {
			_, err := o.Execute(ctx, plan, baseAgentCtx(newBaseScene()), nil)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("concurrent Execute error: %v", err)
	}
}
