package cost

import (
	"time"

	"github.com/google/uuid"
)

type ModelTier string

const (
	TierSonnet ModelTier = "sonnet"
	TierHaiku  ModelTier = "haiku"
	TierLocal  ModelTier = "local"
)

type GenerationCost struct {
	ID           uuid.UUID  `json:"id"`
	GenerationID uuid.UUID  `json:"generation_id"`
	StoryID      uuid.UUID  `json:"story_id"`
	SceneID      uuid.UUID  `json:"scene_id,omitempty"`
	Model        ModelTier  `json:"model"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	TotalTokens  int        `json:"total_tokens"`
	CostUSD      float64    `json:"cost_usd"`
	DurationMs   int64      `json:"duration_ms"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CostTracker struct {
	store Store
}

func NewCostTracker(store Store) *CostTracker {
	return &CostTracker{store: store}
}

func (t *CostTracker) Record(generationID, storyID, sceneID uuid.UUID, model ModelTier, inputTokens, outputTokens int, durationMs int64) (*GenerationCost, error) {
	cost := &GenerationCost{
		ID:           uuid.New(),
		GenerationID: generationID,
		StoryID:      storyID,
		SceneID:      sceneID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		CostUSD:      calculateCost(model, inputTokens, outputTokens),
		DurationMs:   durationMs,
		CreatedAt:    time.Now(),
	}
	if err := t.store.Create(cost); err != nil {
		return nil, err
	}
	return cost, nil
}

func calculateCost(model ModelTier, inputTokens, outputTokens int) float64 {
	var inputRate, outputRate float64
	switch model {
	case TierSonnet:
		inputRate = 3.0 / 1_000_000
		outputRate = 15.0 / 1_000_000
	case TierHaiku:
		inputRate = 0.25 / 1_000_000
		outputRate = 1.25 / 1_000_000
	case TierLocal:
		return 0
	}
	return float64(inputTokens)*inputRate + float64(outputTokens)*outputRate
}

type Store interface {
	Create(cost *GenerationCost) error
	GetByGeneration(generationID uuid.UUID) (*GenerationCost, error)
	GetByStory(storyID uuid.UUID) ([]GenerationCost, error)
	GetTotalByStory(storyID uuid.UUID) (totalTokens int, totalCost float64, err error)
}
