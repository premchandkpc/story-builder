package story

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerationContext is a pinned input snapshot. Providers must not fetch mutable state.
type GenerationContext struct {
	Scene             Scene            `json:"scene"`
	CanonVersion      int64            `json:"canonVersion"`
	CharacterRevision int64            `json:"characterRevision"`
	GraphVersion      int64            `json:"graphVersion"`
	Parents           []Scene          `json:"parents"`
	Characters        []Character      `json:"characters"`
	States            []CharacterState `json:"states"`
	GeneratorVersion  string           `json:"generatorVersion"`
}

func (s *Service) generationContext(ctx context.Context, scene Scene) (GenerationContext, error) {
	current, err := s.repo.GetStory(ctx, scene.StoryID)
	if err != nil {
		return GenerationContext{}, err
	}
	input := GenerationContext{Scene: scene, CanonVersion: current.CanonVersion, CharacterRevision: current.CharacterRevision, GraphVersion: current.GraphVersion}
	input.Characters, err = s.repo.ListCharacters(ctx, scene.StoryID)
	if err != nil {
		return input, err
	}
	input.States, err = s.ReplayCharacterStates(ctx, scene.StoryID)
	if err != nil {
		return input, err
	}
	edges, err := s.repo.ListEdges(ctx, scene.StoryID)
	if err != nil {
		return input, err
	}
	for _, edge := range edges {
		if edge.ToSceneID == scene.ID {
			parent, e := s.repo.GetScene(ctx, edge.FromSceneID)
			if e != nil {
				return input, e
			}
			input.Parents = append(input.Parents, parent)
		}
	}
	sort.Slice(input.Parents, func(i, j int) bool { return input.Parents[i].ID < input.Parents[j].ID })
	sort.Slice(input.Characters, func(i, j int) bool { return input.Characters[i].CharacterID < input.Characters[j].CharacterID })
	latest, err := s.repo.GetStory(ctx, scene.StoryID)
	if err != nil {
		return input, err
	}
	if latest.CanonVersion != current.CanonVersion || latest.CharacterRevision != current.CharacterRevision || latest.GraphVersion != current.GraphVersion {
		return input, ErrConflict
	}
	// Layout changes do not alter narrative context or invalidate cached prose.
	input.Scene.Position = Position{}
	input.Scene.Revision = 0
	input.Scene.UpdatedAt = time.Time{}
	for i := range input.Parents {
		input.Parents[i].Position = Position{}
		input.Parents[i].Revision = 0
		input.Parents[i].UpdatedAt = time.Time{}
	}
	return input, nil
}
func (s *Service) GenerateDraft(ctx context.Context, sceneID string, generator ProseGenerator, send func(string) error) (Draft, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		return Draft{}, err
	}
	// Reject overlapping requests rather than queueing uncancellable mutex waiters.
	if _, loaded := s.generationMu.LoadOrStore(scene.StoryID, struct{}{}); loaded {
		return Draft{}, ErrDraftInProgress
	}
	defer s.generationMu.Delete(scene.StoryID)
	if scene.AcceptedDraftID != "" {
		return Draft{}, ErrSceneLocked
	}
	input, err := s.generationContext(ctx, scene)
	if err != nil {
		return Draft{}, err
	}
	input.GeneratorVersion = generator.Version()
	encoded, err := json.Marshal(input)
	if err != nil {
		return Draft{}, err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(encoded))
	if existing, e := s.repo.FindDraft(ctx, scene.ID, hash); e == nil {
		if e = send(existing.Prose); e != nil {
			return Draft{}, e
		}
		return existing, nil
	} else if !errors.Is(e, ErrNotFound) {
		return Draft{}, e
	}
	now := s.now().UTC()
	draft := Draft{Input: &input, ID: s.newID(), StoryID: scene.StoryID, SceneID: scene.ID, Status: DraftGenerating, BaseCanonVersion: input.CanonVersion, BaseSceneRevision: scene.ContentRevision, BaseCharacterRevision: input.CharacterRevision, BaseGraphVersion: input.GraphVersion, ContextHash: hash, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour), LeaseUntil: now.Add(3 * time.Minute)}
	if err := s.repo.CreateDraft(ctx, draft); err != nil {
		return Draft{}, err
	}
	completed := false
	defer func() {
		if !completed {
			draft.Status = DraftFailed
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.repo.UpdateDraft(cleanup, draft)
		}
	}()
	// Give the provider an independent copy so it cannot mutate stored provenance.
	var providerInput GenerationContext
	if err := json.Unmarshal(encoded, &providerInput); err != nil {
		return Draft{}, err
	}
	var prose strings.Builder
	if err := generator.Generate(ctx, providerInput, func(chunk string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if prose.Len()+len(chunk) > 1<<20 {
			return fmt.Errorf("generated prose exceeds 1 MiB")
		}
		prose.WriteString(chunk)
		return send(chunk)
	}); err != nil {
		return draft, err
	}
	if err := ctx.Err(); err != nil {
		return draft, err
	}
	draft.Prose = prose.String()
	draft.Status = DraftReady
	// The deterministic provider proposes no facts. Acceptance records scene_accepted.
	if err := s.repo.UpdateDraft(ctx, draft); err != nil {
		return Draft{}, err
	}
	completed = true
	return draft, nil
}
