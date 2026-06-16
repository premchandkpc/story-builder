package river

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/ledger"
)

// ── DB-backed SceneContextProvider ──────────────────────────

type dbSceneProvider struct {
	q *db.Queries
}

func NewSceneContextProvider(q *db.Queries) SceneContextProvider {
	return &dbSceneProvider{q: q}
}

func (p *dbSceneProvider) CharacterLatest(ctx context.Context, id uuid.UUID) (*canon.Character, error) {
	c, err := p.q.GetCharacterLatest(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	var traits []string
	if err := json.Unmarshal(c.Traits, &traits); err != nil {
		slog.Warn("unmarshal traits", "id", id, "error", err)
	}
	var rels map[string]string
	if err := json.Unmarshal(c.Relationships, &rels); err != nil {
		slog.Warn("unmarshal relationships", "id", id, "error", err)
	}
	return &canon.Character{
		ID:            id,
		Version:       int(c.Version),
		Name:          c.Name,
		Persona:       c.Persona,
		Backstory:     c.Backstory,
		Traits:        traits,
		VoiceSamples:  c.VoiceSamples,
		Relationships: rels,
	}, nil
}

func (p *dbSceneProvider) LocationLatest(ctx context.Context, id uuid.UUID) (*canon.Location, error) {
	loc, err := p.q.GetLocationLatest(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	var props []string
	if err := json.Unmarshal(loc.Props, &props); err != nil {
		slog.Warn("unmarshal props", "id", id, "error", err)
	}
	return &canon.Location{
		ID:          id,
		Version:     int(loc.Version),
		Name:        loc.Name,
		Description: loc.Description,
		Props:       props,
	}, nil
}

func (p *dbSceneProvider) LoreByTags(ctx context.Context, tags []string) ([]canon.Lore, error) {
	rows, err := p.q.SearchLoreByTags(ctx, tags)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(rows))
	for i, r := range rows {
		result[i] = canon.Lore{
			ID:      db.FromUUID(r.ID),
			Tags:    r.Tags,
			Content: r.Content,
		}
	}
	return result, nil
}

func (p *dbSceneProvider) StateByScene(ctx context.Context, storyID, sceneID uuid.UUID) (map[uuid.UUID]ledger.CharacterState, error) {
	rows, err := p.q.GetStatesForScene(ctx, db.GetStatesForSceneParams{
		StoryID:   db.ToUUID(storyID),
		AsOfScene: db.ToUUID(sceneID),
	})
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]ledger.CharacterState, len(rows))
	for _, s := range rows {
		var cs ledger.CharacterState
		if json.Unmarshal(s.State, &cs) == nil {
			charID := db.FromUUID(s.CharacterID)
			result[charID] = cs
		}
	}
	return result, nil
}

func (p *dbSceneProvider) SummaryByLevel(ctx context.Context, storyID uuid.UUID) (string, error) {
	summary, err := p.q.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{
		StoryID: db.ToUUID(storyID),
		Level:   "scene",
	})
	if err != nil {
		return "", err
	}
	return summary.Content, nil
}

// ── DB-backed GenerationWriter ──────────────────────────────

type dbGenWriter struct {
	q *db.Queries
}

func NewGenerationWriter(q *db.Queries) GenerationWriter {
	return &dbGenWriter{q: q}
}

func (w *dbGenWriter) UpdateOutput(ctx context.Context, genID uuid.UUID, output, model string) error {
	return w.q.UpdateGenerationOutput(ctx, db.ToUUID(genID), output, model)
}

// ── DB-backed CharacterNamer ────────────────────────────────

type dbCharNamer struct {
	q *db.Queries
}

func NewCharacterNamer(q *db.Queries) CharacterNamer {
	return &dbCharNamer{q: q}
}

func (n *dbCharNamer) NameByID(ctx context.Context, id uuid.UUID) (string, error) {
	c, err := n.q.GetCharacterLatest(ctx, db.ToUUID(id))
	if err != nil {
		return "", err
	}
	return c.Name, nil
}

// ── DB-backed CharacterStateWriter ──────────────────────────

type dbStateWriter struct {
	q *db.Queries
}

func NewCharacterStateWriter(q *db.Queries) CharacterStateWriter {
	return &dbStateWriter{q: q}
}

func (w *dbStateWriter) UpsertState(ctx context.Context, storyID, characterID, asOfScene uuid.UUID, state ledger.CharacterState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return w.q.UpsertCharacterState(ctx, db.UpsertCharacterStateParams{
		StoryID:     db.ToUUID(storyID),
		CharacterID: db.ToUUID(characterID),
		AsOfScene:   db.ToUUID(asOfScene),
		State:       data,
	})
}

// ── DB-backed SummaryWriter ─────────────────────────────────

type dbSummaryWriter struct {
	q *db.Queries
}

func NewSummaryWriter(q *db.Queries) SummaryWriter {
	return &dbSummaryWriter{q: q}
}

func (w *dbSummaryWriter) UpsertSceneSummary(ctx context.Context, storyID, sceneID uuid.UUID, content string) error {
	return w.q.UpsertSceneSummary(ctx, db.UpsertSceneSummaryParams{
		StoryID: db.ToUUID(storyID),
		SceneID: db.ToUUID(sceneID),
		Content: content,
	})
}

func (w *dbSummaryWriter) UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error {
	return w.q.UpsertStorySummary(ctx, db.UpsertStorySummaryParams{
		StoryID: db.ToUUID(storyID),
		Content: content,
	})
}

// ── DB-backed ValidationWriter ──────────────────────────────

type dbValWriter struct {
	q *db.Queries
}

func NewValidationWriter(q *db.Queries) ValidationWriter {
	return &dbValWriter{q: q}
}

func (w *dbValWriter) UpdateValidation(ctx context.Context, genID uuid.UUID, data []byte) error {
	return w.q.UpdateGenerationValidation(ctx, db.ToUUID(genID), data)
}

// ── DB-backed StoryFactory ──────────────────────────────────

type dbStoryFactory struct {
	q *db.Queries
}

func NewStoryFactory(q *db.Queries) StoryFactory {
	return &dbStoryFactory{q: q}
}

func (f *dbStoryFactory) CreateStory(ctx context.Context, title string) (uuid.UUID, error) {
	s, err := f.q.CreateStory(ctx, title)
	if err != nil {
		return uuid.Nil, err
	}
	return db.FromUUID(s.ID), nil
}

func (f *dbStoryFactory) UpdateTitle(ctx context.Context, storyID uuid.UUID, title string) error {
	return f.q.UpdateStoryTitle(ctx, db.ToUUID(storyID), title)
}

func (f *dbStoryFactory) CreateCharacter(ctx context.Context, name, persona, backstory, alignment string, personality, flaws, goals, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error) {
	c, err := f.q.CreateCharacter(ctx, db.CreateCharacterParams{
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: alignment,
		Personality:    db.JSONBytes(personality),
		Flaws:          db.JSONBytes(flaws),
		Goals:          db.JSONBytes(goals),
		Traits:         db.JSONBytes(traits),
		VoiceSamples:   voiceSamples,
		Relationships:  db.JSONBytes(relationships),
		ParentID:       db.ToUUID(uuid.Nil),
	})
	if err != nil {
		return nil, err
	}
	return &canon.Character{
		ID:            db.FromUUID(c.ID),
		Name:          c.Name,
		Persona:       c.Persona,
		Backstory:     c.Backstory,
		Traits:        traits,
		VoiceSamples:  voiceSamples,
		Relationships: relationships,
	}, nil
}

func (f *dbStoryFactory) ListChapters(ctx context.Context, storyID uuid.UUID) ([]graph.Chapter, error) {
	rows, err := f.q.ListChapters(ctx, db.ToUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Chapter, len(rows))
	for i, r := range rows {
		result[i] = graph.Chapter{
			ID:         db.FromUUID(r.ID),
			StoryID:    db.FromUUID(r.StoryID),
			Title:      r.Title,
			Goal:       r.Goal,
			Summary:    r.Summary,
			Status:     graph.ChapterStatus(r.Status),
			OrderIndex: int(r.OrderIndex),
		}
	}
	return result, nil
}

func (f *dbStoryFactory) CreateChapter(ctx context.Context, storyID uuid.UUID, title string, orderIndex int) error {
	_, err := f.q.CreateChapter(ctx, db.CreateChapterParams{
		StoryID:    db.ToUUID(storyID),
		Title:      title,
		OrderIndex: int32(orderIndex),
	})
	return err
}

func (f *dbStoryFactory) CreateScene(ctx context.Context, chapterID, storyID uuid.UUID, beatIntent, pov, tone string, targetWords int, charRefs []uuid.UUID) (uuid.UUID, error) {
	refs := make([]pgtype.UUID, len(charRefs))
	for i, ref := range charRefs {
		refs[i] = db.ToUUID(ref)
	}
	s, err := f.q.CreateScene(ctx, db.CreateSceneParams{
		ChapterID:     db.ToUUID(chapterID),
		StoryID:       db.ToUUID(storyID),
		BeatIntent:    beatIntent,
		CharacterRefs: refs,
		Pov:           pov,
		Tone:          tone,
		TargetWords:   int32(targetWords),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return db.FromUUID(s.ID), nil
}

func (f *dbStoryFactory) CreateEdge(ctx context.Context, storyID, fromScene, toScene uuid.UUID, edgeType string) error {
	return f.q.CreateSceneEdge(ctx, db.CreateSceneEdgeParams{
		StoryID:   db.ToUUID(storyID),
		FromScene: db.ToUUID(fromScene),
		ToScene:   db.ToUUID(toScene),
		EdgeType:  edgeType,
	})
}
