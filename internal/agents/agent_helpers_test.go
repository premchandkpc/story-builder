package agents

import (
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func TestResolveCharID_fromPayload(t *testing.T) {
	in := AgentInput{
		Payload: map[string]any{"characterId": "char_2"},
		Ctx:     &AgentContext{ParticipantIDs: []string{"char_1", "char_2"}},
	}
	if got := resolveCharID(in); got != "char_2" {
		t.Errorf("resolveCharID = %q, want %q", got, "char_2")
	}
}

func TestResolveCharID_singleParticipant(t *testing.T) {
	in := AgentInput{
		Ctx: &AgentContext{ParticipantIDs: []string{"char_1"}},
	}
	if got := resolveCharID(in); got != "char_1" {
		t.Errorf("resolveCharID = %q, want %q", got, "char_1")
	}
}

func TestResolveCharID_empty(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{}}
	if got := resolveCharID(in); got != "" {
		t.Errorf("resolveCharID = %q, want empty", got)
	}
}

func TestResolveCharID_rotates(t *testing.T) {
	in := AgentInput{
		Ctx: &AgentContext{
			ParticipantIDs: []string{"char_a", "char_b"},
			Turns: []*domain.SceneTurn{
				{Role: "character", Number: 1},
			},
		},
	}
	if got := resolveCharID(in); got != "char_b" {
		t.Errorf("resolveCharID = %q, want %q (second participant)", got, "char_b")
	}
}

func TestExtractFact_empty(t *testing.T) {
	if got := extractFact(llm.StateDelta{Character: "bob"}); got != "" {
		t.Errorf("extractFact = %q, want empty", got)
	}
}

func TestExtractFact_mood(t *testing.T) {
	got := extractFact(llm.StateDelta{Character: "bob", Mood: "angry"})
	want := "bob: mood: angry"
	if got != want {
		t.Errorf("extractFact = %q, want %q", got, want)
	}
}

func TestExtractFact_full(t *testing.T) {
	got := extractFact(llm.StateDelta{
		Character: "alice", Mood: "hopeful", NewLocation: "garden",
		Learned: []string{"truth"}, ItemsGained: []string{"key"},
	})
	if got == "" {
		t.Fatal("extractFact returned empty")
	}
}

func TestBuildRoster(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		Characters: []*domain.Character{
			{CharID: "c1", Name: "Alice"},
			{CharID: "c2", Name: "Bob"},
		},
	}}
	r := buildRoster(in)
	if len(r) != 2 {
		t.Errorf("roster len = %d, want 2", len(r))
	}
	if r["c1"] != "Alice" {
		t.Errorf("roster[c1] = %q, want Alice", r["c1"])
	}
}

func TestBuildSceneText(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		Scene: &domain.Scene{Title: "Test Scene", BeatIntent: "show conflict"},
		Turns: []*domain.SceneTurn{
			{Role: "character", Output: "Hello!"},
		},
	}}
	text := buildSceneText(in)
	if text == "" {
		t.Fatal("buildSceneText returned empty")
	}
}

func TestRunRuleChecks_health(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		CharStates: []*domain.CharacterState{
			{CharacterID: "c1", Health: -5},
		},
	}}
	v := runRuleChecks(in)
	found := false
	for _, vi := range v {
		if vi["type"] == "invalid_health" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runRuleChecks should flag negative health")
	}
}

func TestRunRuleChecks_timeline(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		Timeline: []*domain.TimelineEvent{
			{SceneID: "s1", Order: 2},
			{SceneID: "s2", Order: 1},
		},
	}}
	v := runRuleChecks(in)
	found := false
	for _, vi := range v {
		if vi["type"] == "timeline_order" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runRuleChecks should flag timeline order violation")
	}
}

func TestRunRuleChecks_locationJump(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		CharStates: []*domain.CharacterState{
			{CharacterID: "c1", Location: "room_a"},
			{CharacterID: "c1", Location: "room_b"},
		},
	}}
	v := runRuleChecks(in)
	found := false
	for _, vi := range v {
		if vi["type"] == "location_jump" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runRuleChecks should flag location jump")
	}
}

func TestRunRuleChecks_lowConfidence(t *testing.T) {
	in := AgentInput{Ctx: &AgentContext{
		CanonDeltas: []*domain.CanonDelta{
			{Fact: "unlikely event", Confidence: 0.2},
		},
	}}
	v := runRuleChecks(in)
	found := false
	for _, vi := range v {
		if vi["type"] == "low_confidence_delta" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runRuleChecks should flag low confidence delta")
	}
}
