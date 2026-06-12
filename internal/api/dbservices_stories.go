package api

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
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/river"
	riv "github.com/riverqueue/river"
)

type dbGraphStoryService struct{ q *db.Queries }

func NewDBGraphStoryService(q *db.Queries) *dbGraphStoryService {
	return &dbGraphStoryService{q: q}
}

func (s *dbGraphStoryService) Create(ctx context.Context, title string) (*graph.Story, error) {
	st, err := s.q.CreateStory(ctx, title)
	if err != nil {
		return nil, err
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: make(map[string]interface{}),
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbGraphStoryService) Get(ctx context.Context, id uuid.UUID) (*graph.Story, error) {
	st, err := s.q.GetStory(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	var pins map[string]interface{}
	json.Unmarshal(st.CanonPins, &pins)
	if pins == nil {
		pins = make(map[string]interface{})
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: pins,
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbGraphStoryService) List(ctx context.Context) ([]graph.Story, error) {
	stories, err := s.q.ListStories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]graph.Story, len(stories))
	for i, st := range stories {
		result[i] = graph.Story{
			ID:        fromUUID(st.ID),
			Title:     st.Title,
			CanonPins: make(map[string]interface{}),
			CreatedAt: st.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbGraphStoryService) CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateEdge(ctx, db.CreateEdgeParams{
		StoryID:  toUUID(storyID),
		FromNode: toUUID(fromNode),
		ToNode:   toUUID(toNode),
		EdgeType: edgeType,
	})
}

func (s *dbGraphStoryService) ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	edges, err := s.q.ListEdges(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Edge, len(edges))
	for i, e := range edges {
		result[i] = graph.Edge{
			StoryID:  fromUUID(e.StoryID),
			FromNode: fromUUID(e.FromNode),
			ToNode:   fromUUID(e.ToNode),
			EdgeType: graph.EdgeType(e.EdgeType),
		}
	}
	return result, nil
}

func (s *dbGraphStoryService) GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return getNode(ctx, s.q, id)
}

func (s *dbGraphStoryService) ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(ctx, s.q, storyID)
}

func (s *dbGraphStoryService) TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := listNodes(ctx, s.q, storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return graph.TopologicalSort(nodes, edges)
}

type dbGraphNodeService struct{ q *db.Queries }

func NewDBGraphNodeService(q *db.Queries) *dbGraphNodeService {
	return &dbGraphNodeService{q: q}
}

func (s *dbGraphNodeService) Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	n, err := s.q.CreateNode(ctx, db.CreateNodeParams{
		StoryID:       toUUID(storyID),
		BeatIntent:    beatIntent,
		CharacterRefs: refs,
		LocationRef:   locRef,
		Pov:           pov,
		Tone:          tone,
		TargetWords:   int32(targetWords),
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbGraphNodeService) Get(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return getNode(ctx, s.q, id)
}

func (s *dbGraphNodeService) Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	ssBytes := jsonBytes(graph.SceneStructure{FlowType: graph.FlowMonologue, SituationFlow: ""})
	if sceneStructure != nil {
		ssBytes = jsonBytes(sceneStructure)
	}
	n, err := s.q.UpdateNode(ctx, db.UpdateNodeParams{
		ID:             toUUID(id),
		BeatIntent:     beatIntent,
		CharacterRefs:  refs,
		LocationRef:    locRef,
		Pov:            pov,
		Tone:           tone,
		TargetWords:    int32(targetWords),
		SceneStructure: ssBytes,
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbGraphNodeService) SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateNodeSceneStructure(ctx, db.UpdateNodeSceneStructureParams{
		ID:             toUUID(id),
		SceneStructure: jsonBytes(ss),
	})
}

func (s *dbGraphNodeService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(ctx, s.q, storyID)
}

func toDomainNode(n db.Node) *graph.Node {
	refs := make([]uuid.UUID, len(n.CharacterRefs))
	for i, r := range n.CharacterRefs {
		refs[i] = fromUUID(r)
	}
	var locRef *uuid.UUID
	if n.LocationRef.Valid {
		l := fromUUID(n.LocationRef)
		locRef = &l
	}
	var ss *graph.SceneStructure
	if len(n.SceneStructure) > 0 {
		var s graph.SceneStructure
		if json.Unmarshal(n.SceneStructure, &s) == nil {
			ss = &s
		}
	}
	return &graph.Node{
		ID:             fromUUID(n.ID),
		StoryID:        fromUUID(n.StoryID),
		BeatIntent:     n.BeatIntent,
		CharacterRefs:  refs,
		LocationRef:    locRef,
		POV:            n.Pov,
		Tone:           n.Tone,
		TargetWords:    int(n.TargetWords),
		Status:         graph.NodeStatus(n.Status),
		SceneStructure: ss,
		CreatedAt:      n.CreatedAt.Time,
		UpdatedAt:      n.UpdatedAt.Time,
	}
}

func getNode(ctx context.Context, q *db.Queries, id uuid.UUID) (*graph.Node, error) {
	n, err := q.GetNode(ctx, toUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, err
	}
	return toDomainNode(n), nil
}

func listNodes(ctx context.Context, q *db.Queries, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := q.ListNodes(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Node, len(nodes))
	for i, n := range nodes {
		result[i] = *toDomainNode(n)
	}
	return result, nil
}

type dbGenerationService struct {
	q            *db.Queries
	rivClient    *riv.Client[pgx.Tx]
	contextCache *cache.ContextCache
}

func NewDBGenerationService(q *db.Queries, rivClient *riv.Client[pgx.Tx]) *dbGenerationService {
	return &dbGenerationService{q: q, rivClient: rivClient}
}

func NewDBGenerationServiceWithCache(q *db.Queries, rivClient *riv.Client[pgx.Tx], contextCache *cache.ContextCache) *dbGenerationService {
	return &dbGenerationService{q: q, rivClient: rivClient, contextCache: contextCache}
}

func (s *dbGenerationService) Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error) {
	node, err := s.q.GetNode(ctx, toUUID(nodeID))
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	storyID := fromUUID(node.StoryID)

	// Check context cache first
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
