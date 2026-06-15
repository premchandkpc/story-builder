package river

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/ledger"
)

// ── GenerateSceneWorker dependencies ─────────────────────────

type SceneContextProvider interface {
	CharacterLatest(ctx context.Context, id uuid.UUID) (*canon.Character, error)
	LocationLatest(ctx context.Context, id uuid.UUID) (*canon.Location, error)
	LoreByTags(ctx context.Context, tags []string) ([]canon.Lore, error)
	StateByScene(ctx context.Context, storyID, sceneID uuid.UUID) (map[uuid.UUID]ledger.CharacterState, error)
	SummaryByLevel(ctx context.Context, storyID uuid.UUID) (string, error)
}

type GenerationWriter interface {
	UpdateOutput(ctx context.Context, genID uuid.UUID, output, model string) error
}

// ── ExtractStateWorker dependencies ──────────────────────────

type CharacterNamer interface {
	NameByID(ctx context.Context, id uuid.UUID) (string, error)
}

type CharacterStateWriter interface {
	UpsertState(ctx context.Context, storyID, characterID, asOfScene uuid.UUID, state ledger.CharacterState) error
}

// ── UpdateSummaryWorker / MergeBranchesWorker dependencies ───

type SummaryWriter interface {
	UpsertSceneSummary(ctx context.Context, storyID, sceneID uuid.UUID, content string) error
	UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error
}

// ── ValidateSceneWorker dependencies ──────────────────────────

type ValidationWriter interface {
	UpdateValidation(ctx context.Context, genID uuid.UUID, data []byte) error
}

// ── GenerateStoryWorker dependencies ──────────────────────────

type StoryFactory interface {
	CreateStory(ctx context.Context, title string) (uuid.UUID, error)
	UpdateTitle(ctx context.Context, storyID uuid.UUID, title string) error
	CreateCharacter(ctx context.Context, name, persona, backstory, alignment string, personality, flaws, goals, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error)
	ListChapters(ctx context.Context, storyID uuid.UUID) ([]graph.Chapter, error)
	CreateScene(ctx context.Context, chapterID, storyID uuid.UUID, beatIntent, pov, tone string, targetWords int, charRefs []uuid.UUID) (uuid.UUID, error)
	CreateEdge(ctx context.Context, storyID, fromScene, toScene uuid.UUID, edgeType string) error
}
