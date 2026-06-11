package narrative

import (
	"time"

	"github.com/google/uuid"
)

// ── Character Arc ──────────────────────────────────────────────

type ArcType string

const (
	ArcHeroJourney    ArcType = "hero_journey"
	ArcRedemption     ArcType = "redemption"
	ArcFall           ArcType = "fall"
	ArcRise           ArcType = "rise"
	ArcTransformation ArcType = "transformation"
	ArcCustom         ArcType = "custom"
)

type GrowthStage string

const (
	GrowthStasis     GrowthStage = "stasis"
	GrowthTrigger    GrowthStage = "trigger"
	GrowthStruggle   GrowthStage = "struggle"
	GrowthClimax     GrowthStage = "climax"
	GrowthResolution GrowthStage = "resolution"
)

type CharacterArc struct {
	CharacterID   uuid.UUID   `json:"character_id"`
	ArcType       ArcType     `json:"arc_type"`
	StartingState string      `json:"starting_state"`
	CurrentBelief string      `json:"current_belief"`
	FalseBelief   string      `json:"false_belief"`
	CoreWound     string      `json:"core_wound"`
	Fear          string      `json:"fear"`
	Want          string      `json:"want"`
	Need          string      `json:"need"`
	GrowthStage   GrowthStage `json:"growth_stage"`
}

// ── Relationship ───────────────────────────────────────────────

type Relationship struct {
	FromCharacterID uuid.UUID `json:"from_character_id"`
	ToCharacterID   uuid.UUID `json:"to_character_id"`
	Trust           int       `json:"trust"`      // -10..10
	Affection       int       `json:"affection"`  // -10..10
	Fear            int       `json:"fear"`        // 0..10
	Respect         int       `json:"respect"`     // 0..10
	Dependency      int       `json:"dependency"`  // 0..10
	History         string    `json:"history"`
}

// ── Plot Thread ────────────────────────────────────────────────

type ThreadStatus string

const (
	ThreadActive    ThreadStatus = "active"
	ThreadDormant   ThreadStatus = "dormant"
	ThreadResolved  ThreadStatus = "resolved"
	ThreadAbandoned ThreadStatus = "abandoned"
)

type PlotThread struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      ThreadStatus `json:"status"`
}

// ── Act ────────────────────────────────────────────────────────

type Act struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Goal   string `json:"goal"`
	Tone   string `json:"tone"`
}

// ── Story Blueprint ────────────────────────────────────────────

type StoryBlueprint struct {
	ID            uuid.UUID       `json:"id"`
	StoryID       uuid.UUID       `json:"story_id"`
	Premise       string          `json:"premise"`
	Theme         string          `json:"theme"`
	MainConflict  string          `json:"main_conflict"`
	Acts          []Act           `json:"acts"`
	CharacterArcs []CharacterArc  `json:"character_arcs"`
	PlotThreads   []PlotThread    `json:"plot_threads"`
	EndState      string          `json:"end_state"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
