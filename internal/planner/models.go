package planner

import (
	"time"

	"github.com/google/uuid"
)

type PlanStatus string

const (
	PlanDraft     PlanStatus = "draft"
	PlanActive    PlanStatus = "active"
	PlanCompleted PlanStatus = "completed"
	PlanAbandoned PlanStatus = "abandoned"
)

type ChapterPlan struct {
	ID            uuid.UUID       `json:"id"`
	StoryID       uuid.UUID       `json:"story_id"`
	ChapterID     uuid.UUID       `json:"chapter_id"`
	Goal          string          `json:"goal"`
	Conflicts     []string        `json:"conflicts"`
	RequiredScenes []string        `json:"required_scenes"`
	Status        PlanStatus      `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ScenePlan struct {
	ID              uuid.UUID            `json:"id"`
	StoryID         uuid.UUID            `json:"story_id"`
	ChapterID       uuid.UUID            `json:"chapter_id,omitempty"`
	SceneID         uuid.UUID            `json:"scene_id,omitempty"`
	Goal            string               `json:"goal"`
	Conflict        string               `json:"conflict"`
	EmotionShift    map[string]string    `json:"emotion_shift"`
	RelShift        map[string]float64   `json:"relationship_shift"`
	RequiredChars   []uuid.UUID          `json:"required_characters"`
	ExpectedOutcome string               `json:"expected_outcome"`
	RiskFactors     []string             `json:"risk_factors"`
	CanonConstraints []string            `json:"canon_constraints"`
	Status          PlanStatus           `json:"status"`
	CreatedAt       time.Time            `json:"created_at"`
}

type PlannerService interface {
	PlanChapter(storyID, chapterID uuid.UUID, goal string, conflicts []string) (*ChapterPlan, error)
	PlanScene(storyID, chapterID, sceneID uuid.UUID, context SceneContext) (*ScenePlan, error)
	GetChapterPlan(chapterID uuid.UUID) (*ChapterPlan, error)
	GetScenePlan(sceneID uuid.UUID) (*ScenePlan, error)
	UpdateScenePlan(plan *ScenePlan) error
}

type SceneContext struct {
	StoryID         uuid.UUID                      `json:"story_id"`
	ChapterGoal     string                         `json:"chapter_goal"`
	PreviousScene   string                         `json:"previous_scene_outcome,omitempty"`
	Characters      []uuid.UUID                    `json:"characters"`
	CharacterEmotions map[uuid.UUID]string          `json:"character_emotions,omitempty"`
	Relationships   map[string]float64              `json:"key_relationships,omitempty"`
	TimelineEvents  []string                        `json:"recent_timeline_events,omitempty"`
	Tension         float64                         `json:"current_tension"`
	Pacing          string                          `json:"pacing"`
}
