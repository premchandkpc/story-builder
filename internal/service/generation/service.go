package generation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/river"
	riv "github.com/riverqueue/river"
)

type GenerationService interface {
	Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error)
	AcceptGeneration(ctx context.Context, nodeID, genID uuid.UUID) error
	ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error)
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

func (s *memoryGenerationService) Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration -- not implemented in memory mode")
}

func (s *memoryGenerationService) AcceptGeneration(ctx context.Context, nodeID, genID uuid.UUID) error {
	for i := range s.gens {
		if s.gens[i].ID == genID.String() && s.gens[i].NodeID == nodeID.String() {
			s.gens[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found for node %s", genID, nodeID)
}

func (s *memoryGenerationService) ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error) {
	var result []compiler.Generation
	for _, g := range s.gens {
		if g.NodeID == nodeID.String() {
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

func (s *dbGenerationService) Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error) {
	node, err := s.q.GetNode(ctx, toUUID(nodeID))
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	storyID := fromUUID(node.StoryID)

	if s.contextCache != nil {
		if cached, err := s.contextCache.Get(ctx, storyID.String()); err == nil {
			hash, err := cached.Hash()
			if err == nil {
				_ = s.contextCache.SetHash(ctx, storyID.String(), hash)
				return s.createGenerationRecord(ctx, nodeID, storyID, node, cached, hash)
			}
		}
	}

	compiled, err := s.compileContext(ctx, storyID, node)
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

	return s.createGenerationRecord(ctx, nodeID, storyID, node, compiled, hash)
}

func (s *dbGenerationService) createGenerationRecord(ctx context.Context, nodeID uuid.UUID, storyID uuid.UUID, node db.Node, compiled *compiler.CompiledContext, hash string) (*compiler.Generation, error) {
	promptSnapshot := compiled.BuildScenePromptSnapshot()

	dbGen, err := s.q.CreateGeneration(ctx, db.CreateGenerationParams{
		NodeID:         toUUID(nodeID),
		ContextHash:    hash,
		PromptSnapshot: promptSnapshot,
		Output:         "",
		Model:          string(llm.ModelSonnet),
	})
	if err != nil {
		return nil, fmt.Errorf("create generation: %w", err)
	}

	genID := fromUUID(dbGen.ID)

	charRefs := make([]uuid.UUID, len(node.CharacterRefs))
	for i, ref := range node.CharacterRefs {
		charRefs[i] = fromUUID(ref)
	}
	var locRef *uuid.UUID
	if node.LocationRef.Valid {
		lr := fromUUID(node.LocationRef)
		locRef = &lr
	}

	_, err = s.rivClient.Insert(ctx, &river.GenerateSceneArgs{
		StoryID:       storyID,
		NodeID:        nodeID,
		GenID:         genID,
		ContextHash:   hash,
		CharacterRefs: charRefs,
		LocationRef:   locRef,
		BeatIntent:    node.BeatIntent,
		POV:           node.Pov,
		Tone:          node.Tone,
		TargetWords:   int(node.TargetWords),
	}, &riv.InsertOpts{
		Queue: river.QueueGenerate,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue generate: %w", err)
	}

	return &compiler.Generation{
		ID:             genID.String(),
		NodeID:         nodeID.String(),
		ContextHash:    hash,
		PromptSnapshot: promptSnapshot,
		Output:         "",
		Model:          string(llm.ModelSonnet),
	}, nil
}

func (s *dbGenerationService) compileContext(ctx context.Context, storyID uuid.UUID, node db.Node) (*compiler.CompiledContext, error) {
	ctx2 := &compiler.CompiledContext{
		BeatIntent:  node.BeatIntent,
		POV:         node.Pov,
		Tone:        node.Tone,
		TargetWords: int(node.TargetWords),
	}

	var charCards []canon.Card
	for _, ref := range node.CharacterRefs {
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

	if node.LocationRef.Valid {
		loc, err := s.q.GetLocationLatest(ctx, node.LocationRef)
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

	return ctx2, nil
}

func (s *dbGenerationService) AcceptGeneration(ctx context.Context, nodeID, genID uuid.UUID) error {
	if err := s.q.AcceptGeneration(ctx, toUUID(genID)); err != nil {
		return err
	}

	if s.contextCache != nil {
		node, err := s.q.GetNode(ctx, toUUID(nodeID))
		if err == nil {
			storyID := fromUUID(node.StoryID)
			_ = s.contextCache.Invalidate(ctx, storyID.String())
		}
	}
	if err := s.q.RejectOtherGenerations(ctx, db.RejectOtherGenerationsParams{
		NodeID: toUUID(nodeID),
		ID:     toUUID(genID),
	}); err != nil {
		return err
	}

	if s.rivClient == nil {
		return nil
	}

	node, err := s.q.GetNode(ctx, toUUID(nodeID))
	if err != nil {
		return err
	}
	storyID := fromUUID(node.StoryID)

	var charRefs []uuid.UUID
	for _, ref := range node.CharacterRefs {
		charRefs = append(charRefs, fromUUID(ref))
	}

	gens, err := s.q.ListGenerationsForNode(ctx, toUUID(nodeID))
	if err != nil {
		return err
	}
	var acceptedGeneration db.Generation
	for _, g := range gens {
		if fromUUID(g.ID) == genID {
			acceptedGeneration = g
			break
		}
	}
	if acceptedGeneration.ID.Valid {
		_, err = s.rivClient.Insert(ctx, &river.ExtractStateArgs{
			StoryID:       storyID,
			NodeID:        nodeID,
			GenerationID:  genID,
			SceneText:     acceptedGeneration.Output,
			CharacterRefs: charRefs,
		}, &riv.InsertOpts{Queue: river.QueueExtract})
		if err != nil {
			return fmt.Errorf("enqueue extract: %w", err)
		}

		prevSummary := ""
		if summary, err := s.q.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{StoryID: toUUID(storyID), Level: "story"}); err == nil {
			prevSummary = summary.Content
		}
		_, err = s.rivClient.Insert(ctx, &river.UpdateSummaryArgs{
			StoryID:         storyID,
			NodeID:          nodeID,
			PreviousSummary: prevSummary,
			AcceptedScene:   acceptedGeneration.Output,
		}, &riv.InsertOpts{Queue: river.QueueDefault})
		if err != nil {
			return fmt.Errorf("enqueue summary: %w", err)
		}

		_, err = s.rivClient.Insert(ctx, &river.ValidateSceneArgs{
			StoryID:       storyID,
			NodeID:        nodeID,
			GenerationID:  genID,
			CompiledCanon: acceptedGeneration.PromptSnapshot,
			CharState:     "{}",
			SceneText:     acceptedGeneration.Output,
		}, &riv.InsertOpts{Queue: river.QueueValidate})
		if err != nil {
			return fmt.Errorf("enqueue validation: %w", err)
		}
	}

	return nil
}

func (s *dbGenerationService) ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error) {
	gens, err := s.q.ListGenerationsForNode(ctx, toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	result := make([]compiler.Generation, len(gens))
	for i, g := range gens {
		result[i] = compiler.Generation{
			ID:             fromUUID(g.ID).String(),
			NodeID:         fromUUID(g.NodeID).String(),
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
	storyID := fromUUID(story.ID)
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

// ── helpers ──────────────────────────────────────────────────────

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromUUID(u pgtype.UUID) uuid.UUID {
	id, _ := uuid.FromBytes(u.Bytes[:])
	return id
}

func jsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
