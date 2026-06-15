package context

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/memory"
)

type BuilderService interface {
	BuildSceneContext(ctx context.Context, sceneID uuid.UUID) (*compiler.CompiledContext, error)
	BuildContextWithMemories(ctx context.Context, sceneID uuid.UUID, memService MemoryLookup) (*compiler.CompiledContext, error)
}

type MemoryLookup interface {
	RetrieveMemories(ctx context.Context, storyID, characterID uuid.UUID) ([]memory.Memory, error)
}

type builderService struct {
	q *db.Queries
}

func NewBuilderService(q *db.Queries) BuilderService {
	return &builderService{q: q}
}

func (s *builderService) BuildSceneContext(ctx context.Context, sceneID uuid.UUID) (*compiler.CompiledContext, error) {
	scene, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, fmt.Errorf("context: get scene: %w", err)
	}
	return s.build(ctx, scene)
}

func (s *builderService) BuildContextWithMemories(ctx context.Context, sceneID uuid.UUID, memService MemoryLookup) (*compiler.CompiledContext, error) {
	scene, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, fmt.Errorf("context: get scene: %w", err)
	}
	cc, err := s.build(ctx, scene)
	if err != nil {
		return nil, err
	}
	storyID := db.FromUUID(scene.StoryID)
	for _, ref := range scene.CharacterRefs {
		charID := db.FromUUID(ref)
		mems, err := memService.RetrieveMemories(ctx, storyID, charID)
		if err != nil {
			continue
		}
		for _, m := range mems {
			cc.Lore = append(cc.Lore, fmt.Sprintf("[Memory: %s] %s", m.Type, m.Summary))
		}
	}
	return cc, nil
}

func (s *builderService) build(ctx context.Context, scene db.Scene) (*compiler.CompiledContext, error) {
	storyID := db.FromUUID(scene.StoryID)

	cc := &compiler.CompiledContext{
		BeatIntent:  scene.BeatIntent,
		POV:         scene.Pov,
		Tone:        scene.Tone,
		TargetWords: int(scene.TargetWords),
	}

	var charCards []canon.Card
	for _, ref := range scene.CharacterRefs {
		c, err := s.q.GetCharacterLatest(ctx, ref)
		if err != nil {
			continue
		}
		var traits []string
		_ = json.Unmarshal(c.Traits, &traits)
		var rels map[string]string
		_ = json.Unmarshal(c.Relationships, &rels)

		charCards = append(charCards, canon.Card{
			Name:          c.Name,
			Description:   c.Persona,
			Type:          "character",
			Traits:        traits,
			VoiceSamples:  c.VoiceSamples,
			Relationships: rels,
		})
	}
	cc.CharacterCards = charCards

	if scene.LocationRef.Valid {
		loc, err := s.q.GetLocationLatest(ctx, scene.LocationRef)
		if err == nil {
			var props []string
			_ = json.Unmarshal(loc.Props, &props)
			cc.LocationCard = &canon.Card{
				Name:        loc.Name,
				Description: loc.Description,
				Type:        "location",
				Props:       props,
			}
		}
	}

	loreTags := make([]string, len(charCards))
	for i, c := range charCards {
		loreTags[i] = c.Name
	}
	lore, err := s.q.SearchLoreByTags(ctx, loreTags)
	if err == nil {
		for _, l := range lore {
			cc.Lore = append(cc.Lore, l.Content)
		}
	}

	states, err := s.q.GetStatesForScene(ctx, db.GetStatesForSceneParams{
		StoryID:   db.ToUUID(storyID),
		AsOfScene: scene.ID,
	})
	if err == nil {
		cc.CharState = make(map[string]ledger.CharacterState, len(states))
		for _, st := range states {
			var cs ledger.CharacterState
			if json.Unmarshal(st.State, &cs) == nil {
				charID := db.FromUUID(st.CharacterID)
				cc.CharState[charID.String()] = cs
			}
		}
	}

	return cc, nil
}
