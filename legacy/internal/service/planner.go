package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type PlannerService struct {
	StoryRepo    repository.StoryRepository
	SceneRepo    repository.SceneRepository
	EdgeRepo     repository.SceneEdgeRepository
	CharRepo     repository.CharacterRepository
	BlueprintRepo repository.BlueprintRepository
}

type ScenePlan struct {
	SceneID           string
	Purpose           domain.ScenePurpose
	ParticipantIntent map[string]string
	EntryStatePreview string
	SuggestedTone     string
	SuggestedPOV      string
	SuggestedWords    int
}

func NewPlannerService(
	storyRepo repository.StoryRepository,
	sceneRepo repository.SceneRepository,
	edgeRepo repository.SceneEdgeRepository,
	charRepo repository.CharacterRepository,
	bpRepo repository.BlueprintRepository,
) *PlannerService {
	return &PlannerService{
		StoryRepo:    storyRepo,
		SceneRepo:    sceneRepo,
		EdgeRepo:     edgeRepo,
		CharRepo:     charRepo,
		BlueprintRepo: bpRepo,
	}
}

func (p *PlannerService) PlanScene(ctx context.Context, sceneID string) (*ScenePlan, error) {
	scene, err := p.SceneRepo.Get(ctx, sceneID)
	if err != nil || scene == nil {
		return nil, fmt.Errorf("scene not found: %w", err)
	}

	bp, err := p.BlueprintRepo.GetByStory(ctx, scene.StoryID)
	if err != nil {
		bp = nil
	}

	edges, err := p.EdgeRepo.ListByStory(ctx, scene.StoryID)
	if err != nil {
		return nil, err
	}

	var incomingEdges, outgoingEdges []*domain.SceneEdge
	for _, e := range edges {
		if e.ToSceneID == sceneID {
			incomingEdges = append(incomingEdges, e)
		}
		if e.FromSceneID == sceneID {
			outgoingEdges = append(outgoingEdges, e)
		}
	}

	plan := &ScenePlan{
		SceneID:           sceneID,
		SuggestedTone:     scene.Tone,
		SuggestedPOV:      scene.POV,
		SuggestedWords:    scene.TargetWords,
		ParticipantIntent: make(map[string]string),
	}

	plan.Purpose.SceneID = sceneID
	plan.Purpose.StoryID = scene.StoryID
	plan.Purpose.EntryState = make(map[string]string)
	plan.Purpose.ExitState = make(map[string]string)

	if bp != nil {
		totalScenes := countScenes(edges)
		actNumber := estimateAct(scene.TimelinePosition, totalScenes)
		for _, arc := range bp.CharacterArcs {
			if actNumber == 2 {
				plan.Purpose.AdvancingArcs = append(plan.Purpose.AdvancingArcs, arc.CharacterID)
			}
		}
	}

	if scene.BeatIntent != "" {
		plan.Purpose.RequiredBeats = []domain.BeatDef{
			{Type: "confrontation", Description: scene.BeatIntent, Mandatory: true},
		}
	}

	if len(outgoingEdges) > 1 {
		plan.Purpose.ConflictType = "choice"
		plan.Purpose.RequiredBeats = append(plan.Purpose.RequiredBeats, domain.BeatDef{
			Type: "decision", Description: "choose path forward", Mandatory: true,
		})
	}

	for _, charID := range scene.Participants {
		char, err := p.CharRepo.GetLatest(ctx, charID)
		if err == nil && char != nil {
			plan.ParticipantIntent[charID] = char.Persona
		}
	}

	return plan, nil
}

func countScenes(edges []*domain.SceneEdge) int {
	seen := make(map[string]bool)
	for _, e := range edges {
		seen[e.FromSceneID] = true
		seen[e.ToSceneID] = true
	}
	return len(seen)
}

func estimateAct(timelinePosition, totalScenes int) int {
	if totalScenes == 0 {
		return 1
	}
	progress := float64(timelinePosition) / float64(totalScenes)
	switch {
	case progress < 0.3:
		return 1
	case progress < 0.7:
		return 2
	default:
		return 3
	}
}
