package river

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/event"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/validation"
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
	Prose    llm.ProseService
	Provider SceneContextProvider
	GenStore GenerationWriter
}

func NewGenerateSceneWorker(prose llm.ProseService, provider SceneContextProvider, genStore GenerationWriter) *GenerateSceneWorker {
	return &GenerateSceneWorker{Prose: prose, Provider: provider, GenStore: genStore}
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
	return w.GenStore.UpdateOutput(ctx, args.GenID, resp.Content, model)
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
		c, err := w.Provider.CharacterLatest(ctx, ref)
		if err != nil {
			continue
		}
		charCards = append(charCards, canon.Card{
			Name:          c.Name,
			Description:   c.Persona,
			Type:          "character",
			Traits:        c.Traits,
			VoiceSamples:  c.VoiceSamples,
			Relationships: c.Relationships,
		})
	}
	params.CharacterCards = charCards

	if args.LocationRef != nil {
		loc, err := w.Provider.LocationLatest(ctx, *args.LocationRef)
		if err == nil {
			params.LocationCard = &canon.Card{
				Name:        loc.Name,
				Description: loc.Description,
				Type:        "location",
				Props:       loc.Props,
			}
		}
	}

	tags := make([]string, len(charCards))
	for i, c := range charCards {
		tags[i] = c.Name
	}
	lore, err := w.Provider.LoreByTags(ctx, tags)
	if err == nil {
		for _, l := range lore {
			params.Lore = append(params.Lore, l.Content)
		}
	}

	stateByChar, err := w.Provider.StateByScene(ctx, args.StoryID, args.NodeID)
	if err == nil {
		params.CharState = make(map[string]interface{})
		for charID, cs := range stateByChar {
			params.CharState[charID.String()] = cs
		}
	}

	summary, err := w.Provider.SummaryByLevel(ctx, args.StoryID)
	if err == nil {
		params.BranchSummary = summary
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
	Namer   CharacterNamer
	Storer  CharacterStateWriter
	Bus     event.Bus
}

func NewExtractStateWorker(extract llm.ExtractionService, namer CharacterNamer, storer CharacterStateWriter, bus event.Bus) *ExtractStateWorker {
	return &ExtractStateWorker{Extract: extract, Namer: namer, Storer: storer, Bus: bus}
}

func (w *ExtractStateWorker) Work(ctx context.Context, job *river.Job[ExtractStateArgs]) error {
	args := job.Args
	roster := make(map[string]string, len(args.CharacterRefs))
	for _, ref := range args.CharacterRefs {
		name, err := w.Namer.NameByID(ctx, ref)
		if err == nil {
			roster[ref.String()] = name
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

		if err := w.Storer.UpsertState(ctx, args.StoryID, d.Character, args.NodeID, cs); err != nil {
			return fmt.Errorf("upsert character state: %w", err)
		}

		if w.Bus != nil {
			evt := &event.Event{
				Type:        event.EvStateDeltaApplied,
				AggregateID: d.Character,
				StoryID:     args.StoryID,
				SceneID:     args.NodeID,
				CharID:      d.Character,
				Payload: map[string]any{
					"character":    d.Character.String(),
					"new_location": d.NewLocation,
					"learned":      d.Learned,
					"mood":         d.Mood,
					"items_gained": d.ItemsGained,
					"items_lost":   d.ItemsLost,
				},
			}
			if len(d.RelationshipChanges) > 0 {
				rels := make([]map[string]string, len(d.RelationshipChanges))
				for i, r := range d.RelationshipChanges {
					rels[i] = map[string]string{"with": r.With.String(), "change": r.Change}
				}
				evt.Payload["relationship_changes"] = rels
			}
			if err := w.Bus.Publish(evt); err != nil {
				slog.Warn("publish state delta event", "error", err)
			}
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
	Writer  SummaryWriter
}

func NewUpdateSummaryWorker(svc llm.SummaryService, writer SummaryWriter) *UpdateSummaryWorker {
	return &UpdateSummaryWorker{Summary: svc, Writer: writer}
}

func (w *UpdateSummaryWorker) Work(ctx context.Context, job *river.Job[UpdateSummaryArgs]) error {
	args := job.Args
	updated, err := w.Summary.UpdateSummary(ctx, args.PreviousSummary, args.AcceptedScene)
	if err != nil {
		return fmt.Errorf("update summary: %w", err)
	}
	return w.Writer.UpsertSceneSummary(ctx, args.StoryID, args.NodeID, updated)
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
	Merge  llm.MergeService
	Writer SummaryWriter
}

func NewMergeBranchesWorker(svc llm.MergeService, writer SummaryWriter) *MergeBranchesWorker {
	return &MergeBranchesWorker{Merge: svc, Writer: writer}
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
	return w.Writer.UpsertStorySummary(ctx, args.StoryID, summary)
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
	Validate  llm.ValidationService
	Validator validation.ValidatorService
	ValStore  ValidationWriter
}

func NewValidateSceneWorker(svc llm.ValidationService, v validation.ValidatorService, valStore ValidationWriter) *ValidateSceneWorker {
	return &ValidateSceneWorker{Validate: svc, Validator: v, ValStore: valStore}
}

func (w *ValidateSceneWorker) Work(ctx context.Context, job *river.Job[ValidateSceneArgs]) error {
	args := job.Args

	if w.Validator != nil {
		w.Validator.ValidateAgainstCanon(args.StoryID, args.NodeID, args.SceneText, nil)
		w.Validator.ValidateCharacterBehavior(uuid.Nil, args.SceneText, nil)
	}

	result, err := w.Validate.ValidateAgainstCanon(ctx, args.CompiledCanon, args.CharState, args.SceneText)
	if err != nil {
		return fmt.Errorf("validate scene: %w", err)
	}
	data, _ := json.Marshal(result)
	if w.ValStore != nil {
		if err := w.ValStore.UpdateValidation(ctx, args.GenerationID, data); err != nil {
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
	Factory StoryFactory
	Prose   llm.ProseService
}

func NewGenerateStoryWorker(outline llm.OutlineService, factory StoryFactory, prose llm.ProseService) *GenerateStoryWorker {
	return &GenerateStoryWorker{Outline: outline, Factory: factory, Prose: prose}
}

func (w *GenerateStoryWorker) Work(ctx context.Context, job *river.Job[GenerateStoryArgs]) error {
	args := job.Args

	outline, err := w.Outline.GenerateOutline(ctx, args.Synopsis)
	if err != nil {
		return fmt.Errorf("generate outline: %w", err)
	}

	storyID := args.StoryID
	if storyID == uuid.Nil {
		storyID, err = w.Factory.CreateStory(ctx, outline.Title)
		if err != nil {
			return fmt.Errorf("create story: %w", err)
		}
	}
	if err := w.Factory.UpdateTitle(ctx, storyID, outline.Title); err != nil {
		return fmt.Errorf("update story title: %w", err)
	}

	charNameToID := make(map[string]uuid.UUID)
	for _, oc := range outline.Characters {
		c, err := w.Factory.CreateCharacter(ctx, oc.Name, oc.Persona, oc.Backstory, oc.MoralAlignment,
			oc.Personality, oc.Flaws, oc.Goals, nil, oc.VoiceSamples, nil)
		if err != nil {
			return fmt.Errorf("create character %s: %w", oc.Name, err)
		}
		charNameToID[oc.Name] = c.ID
	}

	chapters, err := w.Factory.ListChapters(ctx, storyID)
	if err != nil || len(chapters) == 0 {
		return fmt.Errorf("no chapters for story %s", storyID)
	}
	chapterID := chapters[0].ID

	beatTitleToID := make(map[string]uuid.UUID)
	for _, beat := range outline.Beats {
		charRefs := make([]uuid.UUID, 0, len(beat.CharacterNames))
		for _, name := range beat.CharacterNames {
			if id, ok := charNameToID[name]; ok {
				charRefs = append(charRefs, id)
			}
		}

		sceneID, err := w.Factory.CreateScene(ctx, chapterID, storyID, beat.BeatIntent, beat.POV, beat.Tone, beat.TargetWords, charRefs)
		if err != nil {
			return fmt.Errorf("create scene %s: %w", beat.Title, err)
		}
		beatTitleToID[beat.Title] = sceneID
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
		if err := w.Factory.CreateEdge(ctx, storyID, fromID, toID, edge.Type); err != nil {
			return fmt.Errorf("create edge %s->%s: %w", edge.From, edge.To, err)
		}
	}

	return nil
}

// ── Workers Registry ──────────────────────────────────────────

type Dependencies struct {
	Prose       llm.ProseService
	Extract     llm.ExtractionService
	Summary     llm.SummaryService
	Merge       llm.MergeService
	Validate    llm.ValidationService
	Outline     llm.OutlineService
	Validator   validation.ValidatorService
	Provider    SceneContextProvider
	GenStore    GenerationWriter
	Namer       CharacterNamer
	StateStore  CharacterStateWriter
	SumWriter   SummaryWriter
	ValWriter   ValidationWriter
	StoryFac    StoryFactory
	EventBus    event.Bus
}

func Workers(deps *Dependencies) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewGenerateSceneWorker(deps.Prose, deps.Provider, deps.GenStore))
	river.AddWorker(workers, NewExtractStateWorker(deps.Extract, deps.Namer, deps.StateStore, deps.EventBus))
	river.AddWorker(workers, NewUpdateSummaryWorker(deps.Summary, deps.SumWriter))
	river.AddWorker(workers, NewMergeBranchesWorker(deps.Merge, deps.SumWriter))
	river.AddWorker(workers, NewValidateSceneWorker(deps.Validate, deps.Validator, deps.ValWriter))
	river.AddWorker(workers, NewGenerateStoryWorker(deps.Outline, deps.StoryFac, deps.Prose))
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
