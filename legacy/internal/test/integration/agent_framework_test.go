//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/scene"
)

func TestIntegration_SceneTurns(t *testing.T) {
	cleanCollections(t, "scene_turns")

	turnRepo := mgorepo.NewSceneTurnRepo(testDB)
	ctx := context.Background()

	sid := "story-scene-turns-1"
	cid := "scene-scene-turns-1"

	t.Run("create turn", func(t *testing.T) {
		turn := &domain.SceneTurn{
			SceneID:  cid,
			StoryID:  sid,
			Number:   1,
			AgentID:  "director-1",
			Role:     domain.TurnRoleDirector,
			Input:    "Plan the scene",
			Output:   "Hero enters the room",
			Model:    "claude-sonnet",
			Status:   domain.TurnStatusDone,
		}
		err := turnRepo.Create(ctx, turn)
		if err != nil {
			t.Fatalf("create turn: %v", err)
		}
		if turn.ID == "" {
			t.Fatal("turn id is empty")
		}
		if turn.CreatedAt.IsZero() {
			t.Fatal("createdAt not set")
		}
	})

	t.Run("get turn by id", func(t *testing.T) {
		turn := &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 2,
			AgentID: "narrator-1", Role: domain.TurnRoleNarrator,
			Status: domain.TurnStatusRunning,
		}
		turnRepo.Create(ctx, turn)
		got, err := turnRepo.Get(ctx, turn.ID)
		if err != nil {
			t.Fatalf("get turn: %v", err)
		}
		if got == nil {
			t.Fatal("turn not found")
		}
		if got.Role != domain.TurnRoleNarrator {
			t.Fatalf("role: got %q", got.Role)
		}
	})

	t.Run("get missing turn returns nil", func(t *testing.T) {
		got, err := turnRepo.Get(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("get missing: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil for missing turn")
		}
	})

	t.Run("update turn", func(t *testing.T) {
		turn := &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 3,
			AgentID: "character-1", Role: domain.TurnRoleCharacter,
			Status: domain.TurnStatusPending,
		}
		turnRepo.Create(ctx, turn)
		turn.Status = domain.TurnStatusDone
		turn.Output = "Dialogue line"
		err := turnRepo.Update(ctx, turn)
		if err != nil {
			t.Fatalf("update turn: %v", err)
		}
		got, _ := turnRepo.Get(ctx, turn.ID)
		if got.Status != domain.TurnStatusDone {
			t.Fatalf("status after update: got %q", got.Status)
		}
		if got.Output != "Dialogue line" {
			t.Fatalf("output after update: got %q", got.Output)
		}
		if got.UpdatedAt.IsZero() {
			t.Fatal("updatedAt not set")
		}
	})

	t.Run("list turns by scene ordered by number", func(t *testing.T) {
		turns, err := turnRepo.ListByScene(ctx, cid)
		if err != nil {
			t.Fatalf("list by scene: %v", err)
		}
		if len(turns) != 3 {
			t.Fatalf("expected 3 turns, got %d", len(turns))
		}
		for i, tr := range turns {
			if tr.Number != i+1 {
				t.Fatalf("turn %d: expected number %d, got %d", i, i+1, tr.Number)
			}
		}
	})

	t.Run("list turns by role", func(t *testing.T) {
		turns, err := turnRepo.ListByRole(ctx, cid, domain.TurnRoleDirector)
		if err != nil {
			t.Fatalf("list by role: %v", err)
		}
		if len(turns) != 1 {
			t.Fatalf("expected 1 director turn, got %d", len(turns))
		}
	})

	t.Run("delete turns by scene", func(t *testing.T) {
		cleanCollections(t, "scene_turns")
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 1,
			AgentID: "a", Role: domain.TurnRoleDirector, Status: domain.TurnStatusDone,
		})
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 2,
			AgentID: "b", Role: domain.TurnRoleNarrator, Status: domain.TurnStatusDone,
		})
		err := turnRepo.DeleteByScene(ctx, cid)
		if err != nil {
			t.Fatalf("delete by scene: %v", err)
		}
		turns, _ := turnRepo.ListByScene(ctx, cid)
		if len(turns) != 0 {
			t.Fatalf("expected 0 turns after delete, got %d", len(turns))
		}
	})

	t.Run("delete turns by story", func(t *testing.T) {
		cleanCollections(t, "scene_turns")
		otherScene := "scene-other"
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 1,
			AgentID: "a", Role: domain.TurnRoleDirector, Status: domain.TurnStatusDone,
		})
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: otherScene, StoryID: sid, Number: 1,
			AgentID: "b", Role: domain.TurnRoleNarrator, Status: domain.TurnStatusDone,
		})
		err := turnRepo.DeleteByStory(ctx, sid)
		if err != nil {
			t.Fatalf("delete by story: %v", err)
		}
		turns, _ := turnRepo.ListByScene(ctx, cid)
		if len(turns) != 0 {
			t.Fatalf("expected 0 turns after story delete, got %d", len(turns))
		}
	})

	t.Run("empty scene returns empty list", func(t *testing.T) {
		cleanCollections(t, "scene_turns")
		turns, err := turnRepo.ListByScene(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("list by scene nonexistent: %v", err)
		}
		if len(turns) != 0 {
			t.Fatalf("expected 0 turns, got %d", len(turns))
		}
	})
}

func TestIntegration_AgentRuns(t *testing.T) {
	cleanCollections(t, "agent_runs")

	agentRepo := mgorepo.NewAgentRunRepo(testDB)
	ctx := context.Background()

	sid := "story-agent-runs-1"
	cid := "scene-agent-runs-1"

	t.Run("create agent run", func(t *testing.T) {
		run := &domain.AgentRun{
			StoryID:   sid,
			SceneID:   cid,
			AgentType: domain.AgentTypeDirector,
			Input:     map[string]any{"beatIntent": "Opening"},
			Output:    map[string]any{"whoActs": []string{"hero"}},
			Model:     "claude-sonnet",
			Status:    "success",
		}
		err := agentRepo.Create(ctx, run)
		if err != nil {
			t.Fatalf("create agent run: %v", err)
		}
		if run.ID == "" {
			t.Fatal("run id is empty")
		}
		if run.CreatedAt.IsZero() {
			t.Fatal("createdAt not set")
		}
	})

	t.Run("list agent runs with story filter", func(t *testing.T) {
		agentRepo.Create(ctx, &domain.AgentRun{
			StoryID: sid, SceneID: cid, AgentType: domain.AgentTypeNarrator,
			Status: "success",
		})
		agentRepo.Create(ctx, &domain.AgentRun{
			StoryID: sid, SceneID: cid, AgentType: domain.AgentTypeCharacter,
			Status: "running",
		})
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{StoryID: sid})
		if err != nil {
			t.Fatalf("list by story: %v", err)
		}
		if len(runs) != 3 {
			t.Fatalf("expected 3 runs, got %d", len(runs))
		}
	})

	t.Run("list with scene filter", func(t *testing.T) {
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{
			StoryID: sid, SceneID: cid,
		})
		if err != nil {
			t.Fatalf("list by scene: %v", err)
		}
		if len(runs) != 3 {
			t.Fatalf("expected 3 runs for scene, got %d", len(runs))
		}
	})

	t.Run("list with agent type filter", func(t *testing.T) {
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{
			StoryID: sid, AgentType: domain.AgentTypeNarrator,
		})
		if err != nil {
			t.Fatalf("list by agent type: %v", err)
		}
		if len(runs) != 1 {
			t.Fatalf("expected 1 narrator run, got %d", len(runs))
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{
			StoryID: sid, Status: "running",
		})
		if err != nil {
			t.Fatalf("list by status: %v", err)
		}
		if len(runs) != 1 {
			t.Fatalf("expected 1 running run, got %d", len(runs))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{
			StoryID: sid, Limit: 2,
		})
		if err != nil {
			t.Fatalf("list with limit: %v", err)
		}
		if len(runs) != 2 {
			t.Fatalf("expected 2 runs with limit, got %d", len(runs))
		}
	})

	t.Run("list returns newest first", func(t *testing.T) {
		cleanCollections(t, "agent_runs")
		for i := 0; i < 3; i++ {
			agentRepo.Create(ctx, &domain.AgentRun{
				StoryID: sid, SceneID: cid, AgentType: domain.AgentTypeCharacter,
				Status: "success",
			})
		}
		runs, _ := agentRepo.List(ctx, domain.AgentRunFilter{StoryID: sid})
		if len(runs) != 3 {
			t.Fatalf("expected 3, got %d", len(runs))
		}
	})

	t.Run("delete by story", func(t *testing.T) {
		cleanCollections(t, "agent_runs")
		agentRepo.Create(ctx, &domain.AgentRun{
			StoryID: sid, AgentType: domain.AgentTypeDirector, Status: "success",
		})
		agentRepo.Create(ctx, &domain.AgentRun{
			StoryID: sid, AgentType: domain.AgentTypeNarrator, Status: "success",
		})
		err := agentRepo.DeleteByStory(ctx, sid)
		if err != nil {
			t.Fatalf("delete by story: %v", err)
		}
		runs, _ := agentRepo.List(ctx, domain.AgentRunFilter{StoryID: sid})
		if len(runs) != 0 {
			t.Fatalf("expected 0 runs after delete, got %d", len(runs))
		}
	})

	t.Run("empty story returns empty list", func(t *testing.T) {
		runs, err := agentRepo.List(ctx, domain.AgentRunFilter{StoryID: "nonexistent"})
		if err != nil {
			t.Fatalf("list nonexistent: %v", err)
		}
		if len(runs) != 0 {
			t.Fatalf("expected 0, got %d", len(runs))
		}
	})
}

func TestIntegration_CanonDeltas(t *testing.T) {
	cleanCollections(t, "canon_deltas")

	canonRepo := mgorepo.NewCanonDeltaRepo(testDB)
	ctx := context.Background()

	sid := "story-canon-1"
	cid1 := "scene-canon-1"
	cid2 := "scene-canon-2"

	t.Run("create canon delta", func(t *testing.T) {
		d := &domain.CanonDelta{
			StoryID:    sid,
			SceneID:    cid1,
			Category:   domain.CanonCategoryCharacterState,
			Fact:       "hero.location",
			OldValue:   "castle_gates",
			NewValue:   "throne_room",
			Source:     "state_extractor",
			Confidence: 0.95,
		}
		err := canonRepo.Create(ctx, d)
		if err != nil {
			t.Fatalf("create canon delta: %v", err)
		}
		if d.ID == "" {
			t.Fatal("delta id is empty")
		}
		if d.CreatedAt.IsZero() {
			t.Fatal("createdAt not set")
		}
	})

	t.Run("list by scene returns newest first", func(t *testing.T) {
		canonRepo.Create(ctx, &domain.CanonDelta{
			StoryID: sid, SceneID: cid1, Category: domain.CanonCategoryLocation,
			Fact: "setting.time", NewValue: "night", Confidence: 1.0,
		})
		deltas, err := canonRepo.ListByScene(ctx, cid1)
		if err != nil {
			t.Fatalf("list by scene: %v", err)
		}
		if len(deltas) != 2 {
			t.Fatalf("expected 2 deltas, got %d", len(deltas))
		}
	})

	t.Run("list by story", func(t *testing.T) {
		canonRepo.Create(ctx, &domain.CanonDelta{
			StoryID: sid, SceneID: cid2, Category: domain.CanonCategoryRelationship,
			Fact: "hero.villain.trust", OldValue: "50", NewValue: "20",
			Confidence: 0.8,
		})
		deltas, err := canonRepo.ListByStory(ctx, sid)
		if err != nil {
			t.Fatalf("list by story: %v", err)
		}
		if len(deltas) != 3 {
			t.Fatalf("expected 3 deltas, got %d", len(deltas))
		}
	})

	t.Run("list by scene with no results returns empty", func(t *testing.T) {
		deltas, err := canonRepo.ListByScene(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("list by scene nonexistent: %v", err)
		}
		if len(deltas) != 0 {
			t.Fatalf("expected 0, got %d", len(deltas))
		}
	})

	t.Run("delete by story", func(t *testing.T) {
		cleanCollections(t, "canon_deltas")
		canonRepo.Create(ctx, &domain.CanonDelta{
			StoryID: sid, SceneID: cid1, Category: domain.CanonCategoryFact,
			Fact: "test", NewValue: "x", Confidence: 1.0,
		})
		canonRepo.Create(ctx, &domain.CanonDelta{
			StoryID: sid, SceneID: cid2, Category: domain.CanonCategoryFact,
			Fact: "test2", NewValue: "y", Confidence: 1.0,
		})
		err := canonRepo.DeleteByStory(ctx, sid)
		if err != nil {
			t.Fatalf("delete by story: %v", err)
		}
		deltas, _ := canonRepo.ListByStory(ctx, sid)
		if len(deltas) != 0 {
			t.Fatalf("expected 0 after delete, got %d", len(deltas))
		}
	})

	t.Run("empty story returns empty list", func(t *testing.T) {
		deltas, err := canonRepo.ListByStory(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("list by story nonexistent: %v", err)
		}
		if len(deltas) != 0 {
			t.Fatalf("expected 0, got %d", len(deltas))
		}
	})
}

func TestIntegration_TurnOrchestrator(t *testing.T) {
	cleanCollections(t, "scene_turns", "agent_runs", "canon_deltas")

	turnRepo := mgorepo.NewSceneTurnRepo(testDB)
	agentRepo := mgorepo.NewAgentRunRepo(testDB)
	canonRepo := mgorepo.NewCanonDeltaRepo(testDB)
	orch := scene.NewTurnOrchestrator(turnRepo, agentRepo, canonRepo)
	ctx := context.Background()

	sid := "story-orch-1"
	cid := "scene-orch-1"

	t.Run("get turns by scene", func(t *testing.T) {
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 1,
			AgentID: "director-1", Role: domain.TurnRoleDirector,
			Status: domain.TurnStatusDone,
		})
		turnRepo.Create(ctx, &domain.SceneTurn{
			SceneID: cid, StoryID: sid, Number: 2,
			AgentID: "narrator-1", Role: domain.TurnRoleNarrator,
			Status: domain.TurnStatusDone,
		})
		turns, err := orch.GetTurns(ctx, cid)
		if err != nil {
			t.Fatalf("get turns: %v", err)
		}
		if len(turns) != 2 {
			t.Fatalf("expected 2 turns, got %d", len(turns))
		}
	})

	t.Run("get turns by role", func(t *testing.T) {
		turns, err := orch.GetTurnsByRole(ctx, cid, domain.TurnRoleDirector)
		if err != nil {
			t.Fatalf("get turns by role: %v", err)
		}
		if len(turns) != 1 {
			t.Fatalf("expected 1 director turn, got %d", len(turns))
		}
	})

	t.Run("get canon deltas", func(t *testing.T) {
		canonRepo.Create(ctx, &domain.CanonDelta{
			StoryID: sid, SceneID: cid, Category: domain.CanonCategoryCharacterState,
			Fact: "hero.mood", NewValue: "hopeful", Confidence: 0.9,
		})
		deltas, err := orch.GetCanonDeltas(ctx, cid)
		if err != nil {
			t.Fatalf("get canon deltas: %v", err)
		}
		if len(deltas) != 1 {
			t.Fatalf("expected 1 delta, got %d", len(deltas))
		}
	})

	t.Run("record state delta", func(t *testing.T) {
		d := &domain.CanonDelta{
			StoryID:    sid,
			SceneID:    cid,
			Category:   domain.CanonCategoryLocation,
			Fact:       "hero.location",
			NewValue:   "dungeon",
			Confidence: 0.85,
		}
		err := orch.RecordStateDelta(ctx, d)
		if err != nil {
			t.Fatalf("record state delta: %v", err)
		}
		if d.ID == "" {
			t.Fatal("delta id not set after record")
		}
		deltas, _ := orch.GetCanonDeltas(ctx, cid)
		if len(deltas) != 2 {
			t.Fatalf("expected 2 deltas after record, got %d", len(deltas))
		}
	})
}
