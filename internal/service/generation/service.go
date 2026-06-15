package generation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/river"
	riv "github.com/riverqueue/river"
)

type GenerationService interface {
	Generate(ctx context.Context, sceneID uuid.UUID) (*compiler.Generation, error)
	AcceptGeneration(ctx context.Context, sceneID, genID uuid.UUID) error
	ListGenerations(ctx context.Context, sceneID uuid.UUID) ([]compiler.Generation, error)
}

type StoryGeneratorService interface {
	GenerateStory(ctx context.Context, synopsis string) (*StoryGenerateResult, error)
}

type StoryGenerateResult struct {
	StoryID string `json:"story_id"`
	Status  string `json:"status"`
}

// ── Memory (in-memory / stub) ────────────────────────────────────

type memoryGenerationService struct {
	gens []compiler.Generation
}

func NewMemoryGenerationService() *memoryGenerationService {
	return &memoryGenerationService{}
}

func (s *memoryGenerationService) Generate(ctx context.Context, sceneID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration -- not implemented in memory mode")
}

func (s *memoryGenerationService) AcceptGeneration(ctx context.Context, sceneID, genID uuid.UUID) error {
	for i := range s.gens {
		if s.gens[i].ID == genID.String() && s.gens[i].NodeID == sceneID.String() {
			s.gens[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found for scene %s", genID, sceneID)
}

func (s *memoryGenerationService) ListGenerations(ctx context.Context, sceneID uuid.UUID) ([]compiler.Generation, error) {
	var result []compiler.Generation
	for _, g := range s.gens {
		if g.NodeID == sceneID.String() {
			result = append(result, g)
		}
	}
	return result, nil
}

type memoryStoryGeneratorService struct{}

func NewMemoryStoryGeneratorService() *memoryStoryGeneratorService {
	return &memoryStoryGeneratorService{}
}

func (s *memoryStoryGeneratorService) GenerateStory(ctx context.Context, synopsis string) (*StoryGenerateResult, error) {
	return nil, fmt.Errorf("story generation requires LLM integration -- not implemented in memory mode")
}

// ── DB-backed Generation ─────────────────────────────────────────

type dbGenerationService struct {
	q            *db.Queries
	rivClient    *riv.Client[pgx.Tx]
	contextCache *cache.ContextCache
}

func NewDBGenerationService(q *db.Queries, rivClient *riv.Client[pgx.Tx]) *dbGenerationService {
	return &dbGenerationService{q: q, rivClient: rivClient}
}

func NewDBGenerationServiceWithCache(q *db.Queries, rivClient *riv.Client[pgx.Tx], cc *cache.ContextCache) *dbGenerationService {
	return &dbGenerationService{q: q, rivClient: rivClient, contextCache: cc}
}

func (s *dbGenerationService) Generate(ctx context.Context, sceneID uuid.UUID) (*compiler.Generation, error) {
	scene, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, fmt.Errorf("get scene: %w", err)
	}

	storyID := db.FromUUID(scene.StoryID)

	if s.contextCache != nil {
		if cached, err := s.contextCache.Get(ctx, storyID.String()); err == nil {
			hash, err := cached.Hash()
			if err == nil {
				_ = s.contextCache.SetHash(ctx, storyID.String(), hash)
				return s.createGenerationRecord(ctx, sceneID, storyID, scene, cached, hash)
			}
		}
	}

	compiled, err := s.compileContext(ctx, storyID, scene)
	if err != nil {
		return nil, fmt.Errorf("compile context: %w", err)
	}

	if s.contextCache != nil {
		_ = s.contextCache.Set(ctx, storyID.String(), compiled)
	}

	hash, err := compiled.Hash()
	if err != nil {
		return nil, fmt.Errorf("hash context: %w", err)
	}

	if s.contextCache != nil {
		_ = s.contextCache.SetHash(ctx, storyID.String(), hash)
	}

	return s.createGenerationRecord(ctx, sceneID, storyID, scene, compiled, hash)
}

func (s *dbGenerationService) createGenerationRecord(ctx context.Context, sceneID uuid.UUID, storyID uuid.UUID, scene db.Scene, compiled *compiler.CompiledContext, hash string) (*compiler.Generation, error) {
	promptSnapshot := compiled.BuildScenePromptSnapshot()

	dbGen, err := s.q.CreateGeneration(ctx, db.CreateGenerationParams{
		SceneID:        db.ToUUID(sceneID),
		ContextHash:    hash,
		PromptSnapshot: promptSnapshot,
		Output:         "",
		Model:          string(llm.ModelSonnet),
	})
	if err != nil {
		return nil, fmt.Errorf("create generation: %w", err)
	}

	genID := db.FromUUID(dbGen.ID)

	charRefs := make([]uuid.UUID, len(scene.CharacterRefs))
	for i, ref := range scene.CharacterRefs {
		charRefs[i] = db.FromUUID(ref)
	}
	var locRef *uuid.UUID
	if scene.LocationRef.Valid {
		lr := db.FromUUID(scene.LocationRef)
		locRef = &lr
	}

	_, err = s.rivClient.Insert(ctx, &river.GenerateSceneArgs{
		StoryID:       storyID,
		NodeID:        sceneID,
		GenID:         genID,
		ContextHash:   hash,
		CharacterRefs: charRefs,
		LocationRef:   locRef,
		BeatIntent:    scene.BeatIntent,
		POV:           scene.Pov,
		Tone:          scene.Tone,
		TargetWords:   int(scene.TargetWords),
	}, &riv.InsertOpts{
		Queue: river.QueueGenerate,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue generate: %w", err)
	}

	return &compiler.Generation{
		ID:             genID.String(),
		NodeID:         sceneID.String(),
		ContextHash:    hash,
		PromptSnapshot: promptSnapshot,
		Output:         "",
		Model:          string(llm.ModelSonnet),
	}, nil
}

func (s *dbGenerationService) compileContext(ctx context.Context, storyID uuid.UUID, scene db.Scene) (*compiler.CompiledContext, error) {
	ctx2 := &compiler.CompiledContext{
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
		charCards = append(charCards, canon.Card{
			Name:        c.Name,
			Description: c.Persona,
			Type:        "character",
		})
	}
	ctx2.CharacterCards = charCards

	if scene.LocationRef.Valid {
		loc, err := s.q.GetLocationLatest(ctx, scene.LocationRef)
		if err == nil {
			ctx2.LocationCard = &canon.Card{
				Name:        loc.Name,
				Description: loc.Description,
				Type:        "location",
			}
		}
	}

	loreTags := make([]string, 0)
	for _, cc := range charCards {
		loreTags = append(loreTags, cc.Name)
	}
	lore, err := s.q.SearchLoreByTags(ctx, loreTags)
	if err == nil {
		for _, l := range lore {
			ctx2.Lore = append(ctx2.Lore, l.Content)
		}
	}

	states, err := s.q.GetStatesForScene(ctx, db.GetStatesForSceneParams{
		StoryID:   db.ToUUID(storyID),
		AsOfScene: scene.ID,
	})
	if err == nil {
		ctx2.CharState = make(map[string]ledger.CharacterState, len(states))
		for _, st := range states {
			var cs ledger.CharacterState
			if json.Unmarshal(st.State, &cs) == nil {
				charID := db.FromUUID(st.CharacterID)
				ctx2.CharState[charID.String()] = cs
			}
		}
	}

	return ctx2, nil
}

func (s *dbGenerationService) AcceptGeneration(ctx context.Context, sceneID, genID uuid.UUID) error {
	if err := s.q.AcceptGeneration(ctx, db.ToUUID(genID)); err != nil {
		return err
	}

	if s.contextCache != nil {
		scene, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
		if err == nil {
			storyID := db.FromUUID(scene.StoryID)
			_ = s.contextCache.Invalidate(ctx, storyID.String())
		}
	}
	if err := s.q.RejectOtherGenerations(ctx, db.RejectOtherGenerationsParams{
		SceneID: db.ToUUID(sceneID),
		ID:      db.ToUUID(genID),
	}); err != nil {
		return err
	}

	if s.rivClient == nil {
		return nil
	}

	scene, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return err
	}
	storyID := db.FromUUID(scene.StoryID)

	var charRefs []uuid.UUID
	for _, ref := range scene.CharacterRefs {
		charRefs = append(charRefs, db.FromUUID(ref))
	}

	gens, err := s.q.ListGenerationsForScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return err
	}
	var acceptedGeneration db.Generation
	for _, g := range gens {
		if db.FromUUID(g.ID) == genID {
			acceptedGeneration = g
			break
		}
	}
	if acceptedGeneration.ID.Valid {
		_, err = s.rivClient.Insert(ctx, &river.ExtractStateArgs{
			StoryID:       storyID,
			NodeID:        sceneID,
			GenerationID:  genID,
			SceneText:     acceptedGeneration.Output,
			CharacterRefs: charRefs,
		}, &riv.InsertOpts{Queue: river.QueueExtract})
		if err != nil {
			return fmt.Errorf("enqueue extract: %w", err)
		}

		prevSummary := ""
		if summary, err := s.q.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{StoryID: db.ToUUID(storyID), Level: "story"}); err == nil {
			prevSummary = summary.Content
		}
		_, err = s.rivClient.Insert(ctx, &river.UpdateSummaryArgs{
			StoryID:         storyID,
			NodeID:          sceneID,
			PreviousSummary: prevSummary,
			AcceptedScene:   acceptedGeneration.Output,
		}, &riv.InsertOpts{Queue: river.QueueDefault})
		if err != nil {
			return fmt.Errorf("enqueue summary: %w", err)
		}

		compiled, err := s.compileContext(ctx, storyID, scene)
		if err != nil {
			return fmt.Errorf("compile context for validation: %w", err)
		}

		_, err = s.rivClient.Insert(ctx, &river.ValidateSceneArgs{
			StoryID:       storyID,
			NodeID:        sceneID,
			GenerationID:  genID,
			CompiledCanon: compiled.BuildCanonXML(),
			CharState:     compiled.BuildCharStateXML(),
			SceneText:     acceptedGeneration.Output,
		}, &riv.InsertOpts{Queue: river.QueueValidate})
		if err != nil {
			return fmt.Errorf("enqueue validation: %w", err)
		}
	}

	return nil
}

func (s *dbGenerationService) ListGenerations(ctx context.Context, sceneID uuid.UUID) ([]compiler.Generation, error) {
	gens, err := s.q.ListGenerationsForScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, err
	}
	result := make([]compiler.Generation, len(gens))
	for i, g := range gens {
		result[i] = compiler.Generation{
			ID:             db.FromUUID(g.ID).String(),
			NodeID:         db.FromUUID(g.SceneID).String(),
			ContextHash:    g.ContextHash,
			PromptSnapshot: g.PromptSnapshot,
			Output:         g.Output,
			Model:          g.Model,
			Accepted:       g.Accepted,
		}
	}
	return result, nil
}

// ── DB-backed Story Generator ────────────────────────────────────

type dbStoryGeneratorService struct {
	q         *db.Queries
	rivClient *riv.Client[pgx.Tx]
}

func NewDBStoryGeneratorService(q *db.Queries, rivClient *riv.Client[pgx.Tx]) *dbStoryGeneratorService {
	return &dbStoryGeneratorService{q: q, rivClient: rivClient}
}

func (s *dbStoryGeneratorService) GenerateStory(ctx context.Context, synopsis string) (*StoryGenerateResult, error) {
	story, err := s.q.CreateStory(ctx, "Untitled Story")
	if err != nil {
		return nil, fmt.Errorf("create pending story: %w", err)
	}
	storyID := db.FromUUID(story.ID)
	_, err = s.rivClient.Insert(ctx, &river.GenerateStoryArgs{
		StoryID:  storyID,
		Synopsis: synopsis,
	}, &riv.InsertOpts{
		Queue: river.QueueDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue story generation: %w", err)
	}
	return &StoryGenerateResult{
		StoryID: storyID.String(),
		Status:  "pending",
	}, nil
}
