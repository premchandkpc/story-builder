package river

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/riverqueue/river"
)

// ── Generate Scene ────────────────────────────────────────────

type GenerateSceneArgs struct {
	StoryID       uuid.UUID   `json:"story_id"`
	NodeID        uuid.UUID   `json:"node_id"`
	GenID         uuid.UUID   `json:"gen_id"`
	ContextHash   string      `json:"context_hash"`
	CharacterRefs []uuid.UUID `json:"character_refs"`
	LocationRef   *uuid.UUID  `json:"location_ref"`
	BeatIntent    string      `json:"beat_intent"`
	POV           string      `json:"pov"`
	Tone          string      `json:"tone"`
	TargetWords   int         `json:"target_words"`
}

func (GenerateSceneArgs) Kind() string { return "generate_scene" }

type GenerateSceneWorker struct {
	river.WorkerDefaults[GenerateSceneArgs]
	Prose   llm.ProseService
	Queries *db.Queries
}

func NewGenerateSceneWorker(prose llm.ProseService, q *db.Queries) *GenerateSceneWorker {
	return &GenerateSceneWorker{Prose: prose, Queries: q}
}

func (w *GenerateSceneWorker) Work(ctx context.Context, job *river.Job[GenerateSceneArgs]) error {
	args := job.Args

	params, err := w.compilePromptParams(ctx, args)
	if err != nil {
		return fmt.Errorf("compile prompt params: %w", err)
	}

	resp, err := w.Prose.GenerateScene(ctx, *params)
	if err != nil {
		return fmt.Errorf("generate scene: %w", err)
	}

	model := string(llm.ModelSonnet)
	if resp != nil && resp.Model != "" {
		model = resp.Model
	}
	return w.Queries.UpdateGenerationOutput(ctx, db.ToUUID(args.GenID), resp.Content, model)
}

func (w *GenerateSceneWorker) compilePromptParams(ctx context.Context, args GenerateSceneArgs) (*llm.PromptParams, error) {
	params := &llm.PromptParams{
		BeatIntent:  args.BeatIntent,
		POV:         args.POV,
		Tone:        args.Tone,
		TargetWords: args.TargetWords,
	}

	var charCards []canon.Card
	for _, ref := range args.CharacterRefs {
		c, err := w.Queries.GetCharacterLatest(ctx, db.ToUUID(ref))
		if err != nil {
			continue
		}
		var traits []string
		if err := json.Unmarshal(c.Traits, &traits); err != nil {
			slog.Warn("unmarshal traits for character", "id", ref, "error", err)
		}
		var rels map[string]string
		if err := json.Unmarshal(c.Relationships, &rels); err != nil {
			slog.Warn("unmarshal relationships for character", "id", ref, "error", err)
		}
		charCards = append(charCards, canon.Card{
			Name:          c.Name,
			Description:   c.Persona,
			Type:          "character",
			Traits:        traits,
			VoiceSamples:  c.VoiceSamples,
			Relationships: rels,
		})
	}
	params.CharacterCards = charCards

	if args.LocationRef != nil {
		loc, err := w.Queries.GetLocationLatest(ctx, db.ToUUID(*args.LocationRef))
		if err == nil {
			var props []string
			if err := json.Unmarshal(loc.Props, &props); err != nil {
				slog.Warn("unmarshal props for location", "id", *args.LocationRef, "error", err)
			}
			params.LocationCard = &canon.Card{
				Name:        loc.Name,
				Description: loc.Description,
				Type:        "location",
				Props:       props,
			}
		}
	}

	tags := make([]string, len(charCards))
	for i, c := range charCards {
		tags[i] = c.Name
	}
	lore, err := w.Queries.SearchLoreByTags(ctx, tags)
	if err == nil {
		for _, l := range lore {
			params.Lore = append(params.Lore, l.Content)
		}
	}

	states, err := w.Queries.GetStatesForScene(ctx, db.GetStatesForSceneParams{
		StoryID:   db.ToUUID(args.StoryID),
		AsOfScene: db.ToUUID(args.NodeID),
	})
	if err == nil {
		params.CharState = make(map[string]interface{})
		for _, s := range states {
			var cs ledger.CharacterState
			if json.Unmarshal(s.State, &cs) == nil {
				charID := db.FromUUID(s.CharacterID)
				params.CharState[charID.String()] = cs
			}
		}
	}

	summary, err := w.Queries.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{
		StoryID: db.ToUUID(args.StoryID),
		Level:   "scene",
	})
	if err == nil {
		params.BranchSummary = summary.Content
	}

	return params, nil
}

// ── Extract State ─────────────────────────────────────────────

type ExtractStateArgs struct {
	StoryID       uuid.UUID   `json:"story_id"`
	NodeID        uuid.UUID   `json:"node_id"`
	GenerationID  uuid.UUID   `json:"generation_id"`
	SceneText     string      `json:"scene_text"`
	CharacterRefs []uuid.UUID `json:"character_refs"`
}

func (ExtractStateArgs) Kind() string { return "extract_state" }

type ExtractStateWorker struct {
	river.WorkerDefaults[ExtractStateArgs]
	Extract llm.ExtractionService
	Queries *db.Queries
}

func NewExtractStateWorker(extract llm.ExtractionService, q *db.Queries) *ExtractStateWorker {
	return &ExtractStateWorker{Extract: extract, Queries: q}
}

func (w *ExtractStateWorker) Work(ctx context.Context, job *river.Job[ExtractStateArgs]) error {
	args := job.Args
	roster := make(map[string]string, len(args.CharacterRefs))
	for _, ref := range args.CharacterRefs {
		if w.Queries == nil {
			break
		}
		c, err := w.Queries.GetCharacterLatest(ctx, db.ToUUID(ref))
		if err == nil {
			roster[ref.String()] = c.Name
		}
	}
	result, err := w.Extract.ExtractState(ctx, args.SceneText, roster)
	if err != nil {
		return fmt.Errorf("extract state: %w", err)
	}

	for _, d := range result.Deltas {
		cs := ledger.CharacterState{
			StoryID:     args.StoryID,
			CharacterID: d.Character,
			AsOfNode:    args.NodeID,
			Location:    d.NewLocation,
			Knows:       d.Learned,
			Mood:        d.Mood,
			Items:       d.ItemsGained,
		}
		cs.Relationships = make(map[string]string)
		for _, rel := range d.RelationshipChanges {
			cs.Relationships[rel.With.String()] = rel.Change
		}

		stateJSON, err := json.Marshal(cs)
		if err != nil {
			return fmt.Errorf("marshal character state: %w", err)
		}

		if err := w.Queries.UpsertCharacterState(ctx, db.UpsertCharacterStateParams{
			StoryID:     db.ToUUID(args.StoryID),
			CharacterID: db.ToUUID(d.Character),
			AsOfScene:   db.ToUUID(args.NodeID),
			State:       stateJSON,
		}); err != nil {
			return fmt.Errorf("upsert character state: %w", err)
		}
	}

	return nil
}

// ── Update Summary ────────────────────────────────────────────

type UpdateSummaryArgs struct {
	StoryID         uuid.UUID `json:"story_id"`
	NodeID          uuid.UUID `json:"node_id"`
	PreviousSummary string    `json:"previous_summary"`
	AcceptedScene   string    `json:"accepted_scene"`
}

func (UpdateSummaryArgs) Kind() string { return "update_summary" }

type UpdateSummaryWorker struct {
	river.WorkerDefaults[UpdateSummaryArgs]
	Summary llm.SummaryService
	Queries *db.Queries
}

func NewUpdateSummaryWorker(svc llm.SummaryService, q *db.Queries) *UpdateSummaryWorker {
	return &UpdateSummaryWorker{Summary: svc, Queries: q}
}

func (w *UpdateSummaryWorker) Work(ctx context.Context, job *river.Job[UpdateSummaryArgs]) error {
	args := job.Args
	updated, err := w.Summary.UpdateSummary(ctx, args.PreviousSummary, args.AcceptedScene)
	if err != nil {
		return fmt.Errorf("update summary: %w", err)
	}
	return w.Queries.UpsertSceneSummary(ctx, db.UpsertSceneSummaryParams{
		StoryID: db.ToUUID(args.StoryID),
		SceneID: db.ToUUID(args.NodeID),
		Content: updated,
	})
}

// ── Merge Branches ────────────────────────────────────────────

type MergeBranchesArgs struct {
	StoryID      uuid.UUID `json:"story_id"`
	JoinNodeID   uuid.UUID `json:"join_node_id"`
	SummaryA     string    `json:"summary_a"`
	SummaryB     string    `json:"summary_b"`
	TimelineNote string    `json:"timeline_note"`
}

func (MergeBranchesArgs) Kind() string { return "merge_branches" }

type MergeBranchesWorker struct {
	river.WorkerDefaults[MergeBranchesArgs]
	Merge   llm.MergeService
	Queries *db.Queries
}

func NewMergeBranchesWorker(svc llm.MergeService, q *db.Queries) *MergeBranchesWorker {
	return &MergeBranchesWorker{Merge: svc, Queries: q}
}

func (w *MergeBranchesWorker) Work(ctx context.Context, job *river.Job[MergeBranchesArgs]) error {
	args := job.Args
	result, err := w.Merge.MergeBranches(ctx, args.SummaryA, args.SummaryB, args.TimelineNote)
	if err != nil {
		return fmt.Errorf("merge branches: %w", err)
	}
	summary, _ := result["merged_summary"].(string)
	if summary == "" {
		return nil
	}
	return w.Queries.UpsertStorySummary(ctx, db.UpsertStorySummaryParams{
		StoryID: db.ToUUID(args.StoryID),
		Content: summary,
	})
}

// ── Validate Scene ────────────────────────────────────────────

type ValidateSceneArgs struct {
	StoryID       uuid.UUID `json:"story_id"`
	NodeID        uuid.UUID `json:"node_id"`
	GenerationID  uuid.UUID `json:"generation_id"`
	CompiledCanon string    `json:"compiled_canon"`
	CharState     string    `json:"char_state"`
	SceneText     string    `json:"scene_text"`
}

func (ValidateSceneArgs) Kind() string { return "validate_scene" }

type ValidateSceneWorker struct {
	river.WorkerDefaults[ValidateSceneArgs]
	Validate llm.ValidationService
	Queries  *db.Queries
}

func NewValidateSceneWorker(svc llm.ValidationService, q *db.Queries) *ValidateSceneWorker {
	return &ValidateSceneWorker{Validate: svc, Queries: q}
}

func (w *ValidateSceneWorker) Work(ctx context.Context, job *river.Job[ValidateSceneArgs]) error {
	args := job.Args
	result, err := w.Validate.ValidateAgainstCanon(ctx, args.CompiledCanon, args.CharState, args.SceneText)
	if err != nil {
		return fmt.Errorf("validate scene: %w", err)
	}
	data, _ := json.Marshal(result)
	if w.Queries != nil {
		if err := w.Queries.UpdateGenerationValidation(ctx, db.ToUUID(args.GenerationID), data); err != nil {
			return fmt.Errorf("persist validation: %w", err)
		}
	}
	slog.Info("validation result", "generation_id", args.GenerationID, "result", string(data))
	return nil
}

// ── Generate Story ────────────────────────────────────────────

type GenerateStoryArgs struct {
	StoryID  uuid.UUID `json:"story_id,omitempty"`
	Synopsis string    `json:"synopsis"`
}

func (GenerateStoryArgs) Kind() string { return "generate_story" }

type GenerateStoryWorker struct {
	river.WorkerDefaults[GenerateStoryArgs]
	Outline llm.OutlineService
	Queries *db.Queries
	Prose   llm.ProseService
}

func NewGenerateStoryWorker(outline llm.OutlineService, q *db.Queries, prose llm.ProseService) *GenerateStoryWorker {
	return &GenerateStoryWorker{Outline: outline, Queries: q, Prose: prose}
}

func (w *GenerateStoryWorker) Work(ctx context.Context, job *river.Job[GenerateStoryArgs]) error {
	args := job.Args

	outline, err := w.Outline.GenerateOutline(ctx, args.Synopsis)
	if err != nil {
		return fmt.Errorf("generate outline: %w", err)
	}

	storyID := args.StoryID
	if storyID == uuid.Nil {
		story, err := w.Queries.CreateStory(ctx, outline.Title)
		if err != nil {
			return fmt.Errorf("create story: %w", err)
		}
		storyID = db.FromUUID(story.ID)
	}
	if err := w.Queries.UpdateStoryTitle(ctx, db.ToUUID(storyID), outline.Title); err != nil {
		return fmt.Errorf("update story title: %w", err)
	}

	charNameToID := make(map[string]pgtype.UUID)
	for _, oc := range outline.Characters {
		c, err := w.Queries.CreateCharacter(ctx, db.CreateCharacterParams{
			Name:           oc.Name,
			Persona:        oc.Persona,
			Backstory:      oc.Backstory,
			MoralAlignment: oc.MoralAlignment,
			Personality:    db.JSONBytes(oc.Personality),
			Flaws:          db.JSONBytes(oc.Flaws),
			Goals:          db.JSONBytes(oc.Goals),
			Traits:         db.JSONBytes([]string{}),
			VoiceSamples:   oc.VoiceSamples,
			Relationships:  db.JSONBytes(map[string]string{}),
			ParentID:       pgtype.UUID{Valid: false},
		})
		if err != nil {
			return fmt.Errorf("create character %s: %w", oc.Name, err)
		}
		charNameToID[oc.Name] = c.ID
	}

	chapters, err := w.Queries.ListChapters(ctx, db.ToUUID(storyID))
	if err != nil || len(chapters) == 0 {
		return fmt.Errorf("no chapters for story %s", storyID)
	}
	chapterID := chapters[0].ID

	beatTitleToID := make(map[string]pgtype.UUID)
	for _, beat := range outline.Beats {
		charRefs := make([]pgtype.UUID, 0, len(beat.CharacterNames))
		for _, name := range beat.CharacterNames {
			if id, ok := charNameToID[name]; ok {
				charRefs = append(charRefs, id)
			}
		}

		scene, err := w.Queries.CreateScene(ctx, db.CreateSceneParams{
			ChapterID:     chapterID,
			StoryID:       db.ToUUID(storyID),
			BeatIntent:    beat.BeatIntent,
			CharacterRefs: charRefs,
			Pov:           beat.POV,
			Tone:          beat.Tone,
			TargetWords:   int32(beat.TargetWords),
		})
		if err != nil {
			return fmt.Errorf("create scene %s: %w", beat.Title, err)
		}
		beatTitleToID[beat.Title] = scene.ID
	}

	for _, edge := range outline.Edges {
		fromID, ok := beatTitleToID[edge.From]
		if !ok {
			continue
		}
		toID, ok := beatTitleToID[edge.To]
		if !ok {
			continue
		}
		err := w.Queries.CreateSceneEdge(ctx, db.CreateSceneEdgeParams{
			StoryID:   db.ToUUID(storyID),
			FromScene: fromID,
			ToScene:   toID,
			EdgeType:  edge.Type,
		})
		if err != nil {
			return fmt.Errorf("create edge %s->%s: %w", edge.From, edge.To, err)
		}
	}

	return nil
}

// ── Workers Registry ──────────────────────────────────────────

type Dependencies struct {
	Prose    llm.ProseService
	Extract  llm.ExtractionService
	Summary  llm.SummaryService
	Merge    llm.MergeService
	Validate llm.ValidationService
	Outline  llm.OutlineService
	Queries  *db.Queries
}

func Workers(deps *Dependencies) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewGenerateSceneWorker(deps.Prose, deps.Queries))
	river.AddWorker(workers, NewExtractStateWorker(deps.Extract, deps.Queries))
	river.AddWorker(workers, NewUpdateSummaryWorker(deps.Summary, deps.Queries))
	river.AddWorker(workers, NewMergeBranchesWorker(deps.Merge, deps.Queries))
	river.AddWorker(workers, NewValidateSceneWorker(deps.Validate, deps.Queries))
	river.AddWorker(workers, NewGenerateStoryWorker(deps.Outline, deps.Queries, deps.Prose))
	return workers
}

// ── Queue names ───────────────────────────────────────────────

const (
	QueueDefault  = "default"
	QueueGenerate = "generate"
	QueueExtract  = "extract"
	QueueMerge    = "merge"
	QueueValidate = "validate"
)

type Config struct {
	DatabaseURL string
	Workers     *river.Workers
}

type InsertGenerateSceneParams struct {
	Args        GenerateSceneArgs
	Queue       string
	MaxAttempts int
	SchedFor    *time.Time
}

