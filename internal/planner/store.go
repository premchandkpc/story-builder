package planner

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu           sync.RWMutex
	chapterPlans map[uuid.UUID]*ChapterPlan
	scenePlans   map[uuid.UUID]*ScenePlan
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		chapterPlans: make(map[uuid.UUID]*ChapterPlan),
		scenePlans:   make(map[uuid.UUID]*ScenePlan),
	}
}

type plannerService struct {
	store *MemoryStore
}

func NewPlannerService(store *MemoryStore) PlannerService {
	return &plannerService{store: store}
}

func (s *plannerService) PlanChapter(storyID, chapterID uuid.UUID, goal string, conflicts []string) (*ChapterPlan, error) {
	plan := &ChapterPlan{
		ID:             uuid.New(),
		StoryID:        storyID,
		ChapterID:      chapterID,
		Goal:           goal,
		Conflicts:      conflicts,
		RequiredScenes: deriveRequiredScenes(goal, conflicts),
		Status:         PlanDraft,
		CreatedAt:      time.Now(),
	}
	s.store.mu.Lock()
	s.store.chapterPlans[chapterID] = plan
	s.store.mu.Unlock()
	return plan, nil
}

func (s *plannerService) PlanScene(storyID, chapterID, sceneID uuid.UUID, ctx SceneContext) (*ScenePlan, error) {
	plan := &ScenePlan{
		ID:              uuid.New(),
		StoryID:         storyID,
		ChapterID:       chapterID,
		SceneID:         sceneID,
		Goal:            deriveSceneGoal(ctx),
		Conflict:        deriveConflict(ctx),
		EmotionShift:    deriveEmotionShift(ctx),
		RelShift:        deriveRelShift(ctx),
		RequiredChars:   ctx.Characters,
		ExpectedOutcome: "",
		Status:          PlanDraft,
		CreatedAt:       time.Now(),
	}
	s.store.mu.Lock()
	s.store.scenePlans[sceneID] = plan
	s.store.mu.Unlock()
	return plan, nil
}

func (s *plannerService) GetChapterPlan(chapterID uuid.UUID) (*ChapterPlan, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	plan, ok := s.store.chapterPlans[chapterID]
	if !ok {
		return nil, ErrNotFound
	}
	return plan, nil
}

func (s *plannerService) GetScenePlan(sceneID uuid.UUID) (*ScenePlan, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	plan, ok := s.store.scenePlans[sceneID]
	if !ok {
		return nil, ErrNotFound
	}
	return plan, nil
}

func (s *plannerService) UpdateScenePlan(plan *ScenePlan) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.scenePlans[plan.SceneID] = plan
	return nil
}

func deriveRequiredScenes(goal string, conflicts []string) []string {
	scenes := []string{"setup", "confrontation"}
	if len(conflicts) > 1 {
		scenes = append(scenes, "resolution")
	}
	return scenes
}

func deriveSceneGoal(ctx SceneContext) string {
	if ctx.ChapterGoal != "" {
		return "Advance: " + ctx.ChapterGoal
	}
	return "Progress narrative"
}

func deriveConflict(ctx SceneContext) string {
	if ctx.Tension > 0.7 {
		return "escalating conflict"
	}
	return "emerging tension"
}

func deriveEmotionShift(ctx SceneContext) map[string]string {
	shift := make(map[string]string)
	for cid, em := range ctx.CharacterEmotions {
		shift[cid.String()] = em
	}
	if len(shift) == 0 {
		shift["default"] = "curiosity"
	}
	return shift
}

func deriveRelShift(ctx SceneContext) map[string]float64 {
	return ctx.Relationships
}
