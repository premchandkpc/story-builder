package validation

import (
	"context"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
)

func TestValidatePreGeneration_EmptyScene(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePreGeneration(context.Background(), nil)
	if len(errs) == 0 {
		t.Fatal("expected error for nil scene")
	}
}

func TestValidatePreGeneration_NoBeatIntent(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID: "s1",
	})
	found := false
	for _, e := range errs {
		if e.Field == "beatIntent" && e.Severity == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected error for missing beatIntent")
	}
}

func TestValidatePreGeneration_NoParticipants(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:   "s1",
		BeatIntent: "do something",
	})
	found := false
	for _, e := range errs {
		if e.Field == "participants" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected warning for no participants")
	}
}

type stubCharRepo struct {
	chars []*domain.Character
	err   error
}

func (r *stubCharRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error) {
	return r.chars, r.err
}

type stubLocRepo struct {
	locations []*domain.Location
	err       error
}

func (r *stubLocRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error) {
	return r.locations, r.err
}

func (r *stubLocRepo) GetByName(ctx context.Context, storyID, name string) (*domain.Location, error) {
	for _, l := range r.locations {
		if l.Name == name {
			return l, nil
		}
	}
	return nil, nil
}

func TestValidatePreGeneration_UnknownParticipant(t *testing.T) {
	charRepo := &stubCharRepo{
		chars: []*domain.Character{{Name: "Hero"}},
	}
	v := NewSceneValidator(charRepo, nil)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:      "s1",
		BeatIntent:   "do something",
		Participants: []string{"Hero", "Unknown"},
	})
	found := false
	for _, e := range errs {
		if e.Field == "participants" && e.Severity == "warning" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected warning for unknown participant")
	}
}

func TestValidatePreGeneration_KnownParticipant(t *testing.T) {
	charRepo := &stubCharRepo{
		chars: []*domain.Character{{Name: "Hero"}},
	}
	v := NewSceneValidator(charRepo, nil)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:      "s1",
		BeatIntent:   "do something",
		Participants: []string{"Hero"},
	})
	for _, e := range errs {
		if e.Field == "participants" {
			t.Fatalf("unexpected warning: %s", e.Message)
		}
	}
}

func TestValidatePreGeneration_LocationNotFound(t *testing.T) {
	charRepo := &stubCharRepo{}
	locRepo := &stubLocRepo{}
	v := NewSceneValidator(charRepo, locRepo)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:     "s1",
		BeatIntent:  "do something",
		LocationRef: "Nowhere",
	})
	found := false
	for _, e := range errs {
		if e.Field == "locationRef" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected warning for unknown location")
	}
}

func TestValidatePreGeneration_LocationFound(t *testing.T) {
	charRepo := &stubCharRepo{}
	locRepo := &stubLocRepo{
		locations: []*domain.Location{{Name: "Forest"}},
	}
	v := NewSceneValidator(charRepo, locRepo)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:     "s1",
		BeatIntent:  "do something",
		LocationRef: "Forest",
	})
	for _, e := range errs {
		if e.Field == "locationRef" {
			t.Fatalf("unexpected warning: %s", e.Message)
		}
	}
}

func TestValidatePostGeneration_NilScene(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), nil, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no violations for nil scene, got %d", len(errs))
	}
}

func TestValidatePostGeneration_LocationContinuity(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), &domain.Scene{
		LocationRef: "Castle",
	}, []PostGenerationCheck{
		{CharacterID: "Hero", PreviousLocation: "Forest", NewLocation: "Castle"},
	})
	if len(errs) != 0 {
		t.Fatalf("expected no violations when character moved to scene location, got: %v", errs)
	}
}

func TestValidatePostGeneration_LocationMismatch(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), &domain.Scene{
		LocationRef: "Castle",
	}, []PostGenerationCheck{
		{CharacterID: "Hero", PreviousLocation: "Forest", NewLocation: "Dungeon"},
	})
	found := false
	for _, e := range errs {
		if e.Field == "location_continuity" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected location continuity violation when character went to different location than scene")
	}
}

func TestValidatePostGeneration_KnowledgeRedundancy(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), &domain.Scene{ID: "s1"}, []PostGenerationCheck{
		{CharacterID: "Hero", PreviousKnowledge: []string{"The king is dead"}, Learned: []string{"The king is dead"}},
	})
	found := false
	for _, e := range errs {
		if e.Field == "knowledge_redundancy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected knowledge redundancy violation")
	}
}

func TestValidatePostGeneration_NewKnowledge(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), &domain.Scene{ID: "s1"}, []PostGenerationCheck{
		{CharacterID: "Hero", PreviousKnowledge: []string{"The king is dead"}, Learned: []string{"The queen is the killer"}},
	})
	for _, e := range errs {
		t.Fatalf("unexpected violation: %s", e.Message)
	}
}

func TestValidatePostGeneration_NoSceneLocation(t *testing.T) {
	v := NewSceneValidator(nil, nil)
	errs := v.ValidatePostGeneration(context.Background(), &domain.Scene{ID: "s1"}, []PostGenerationCheck{
		{CharacterID: "Hero", PreviousLocation: "Forest", NewLocation: "Castle"},
	})
	for _, e := range errs {
		t.Fatalf("unexpected violation when scene has no location: %s", e.Message)
	}
}

func TestValidatePreGeneration_AllGood(t *testing.T) {
	charRepo := &stubCharRepo{
		chars: []*domain.Character{{Name: "Hero"}},
	}
	locRepo := &stubLocRepo{
		locations: []*domain.Location{{Name: "Forest"}},
	}
	v := NewSceneValidator(charRepo, locRepo)
	errs := v.ValidatePreGeneration(context.Background(), &domain.Scene{
		StoryID:      "s1",
		BeatIntent:   "explore",
		Participants: []string{"Hero"},
		LocationRef:  "Forest",
	})
	// beatIntent present, participants exist, location exists — only no-participants warning should be absent
	for _, e := range errs {
		t.Fatalf("unexpected violation: [%s] %s: %s", e.Severity, e.Field, e.Message)
	}
}
