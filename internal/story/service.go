package story

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ProseGenerator interface {
	Version() string
	Generate(context.Context, GenerationContext, func(string) error) error
}

type Service struct {
	repo         Repository
	now          func() time.Time
	newID        func() string
	generationMu sync.Map // active story IDs; process-local admission control
}

func (s *Service) CreateCharacter(ctx context.Context, storyID, name string, traits []string) (Character, error) {
	return s.DefineCharacter(ctx, storyID, "", CharacterInput{Name: name, Traits: traits})
}

func (s *Service) ListCharacters(ctx context.Context, storyID string) ([]Character, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		return nil, err
	}
	values, err := s.repo.ListCharacters(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	return values, nil
}
func (s *Service) ReplayCharacterStates(ctx context.Context, storyID string) ([]CharacterState, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		return nil, err
	}
	events, err := s.repo.ListCanonEvents(ctx, storyID)
	if err != nil {
		return nil, err
	}
	characters, err := s.repo.ListCharacters(ctx, storyID)
	if err != nil {
		return nil, err
	}
	states := ProjectCharacterStates(events)
	byID := map[string]CharacterState{}
	for _, state := range states {
		byID[state.CharacterID] = state
	}
	out := make([]CharacterState, 0, len(characters))
	for _, character := range characters {
		state := byID[character.CharacterID]
		state.StoryID = storyID
		state.CharacterID = character.CharacterID
		state.Knowledge = appendUnique(append([]string{}, character.InitialKnowledge...), state.Knowledge)
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CharacterID < out[j].CharacterID })
	return out, nil
}
func (s *Service) AcceptDraft(ctx context.Context, id string) (Scene, error) {
	draft, err := s.repo.GetDraft(ctx, id)
	if err != nil {
		return Scene{}, fmt.Errorf("get draft: %w", err)
	}
	if draft.Status == DraftAccepted {
		return s.repo.AcceptDraft(ctx, id)
	}
	edges, err := s.repo.ListEdges(ctx, draft.StoryID)
	if err != nil {
		return Scene{}, fmt.Errorf("list edges: %w", err)
	}
	for _, edge := range edges {
		if edge.ToSceneID != draft.SceneID {
			continue
		}
		parent, err := s.repo.GetScene(ctx, edge.FromSceneID)
		if err != nil {
			return Scene{}, fmt.Errorf("get parent scene: %w", err)
		}
		if parent.AcceptedDraftID == "" {
			return Scene{}, ErrParentsPending
		}
	}
	value, err := s.repo.AcceptDraft(ctx, id)
	if err != nil {
		return Scene{}, fmt.Errorf("accept draft: %w", err)
	}
	return value, nil
}
func (s *Service) RejectDraft(ctx context.Context, id string) error {
	if err := s.repo.DeleteDraft(ctx, id); err != nil {
		return fmt.Errorf("reject draft: %w", err)
	}
	return nil
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, newID: uuid.NewString}
}

func (s *Service) CreateStory(ctx context.Context, title string) (Story, error) {
	title, err := requiredText(title)
	if err != nil {
		return Story{}, ErrInvalidTitle
	}
	story := Story{ID: s.newID(), Title: title, CreatedAt: s.now().UTC()}
	if err := s.repo.CreateStory(ctx, story); err != nil {
		return Story{}, fmt.Errorf("create story: %w", err)
	}
	return story, nil
}

func (s *Service) ListStories(ctx context.Context) ([]Story, error) {
	stories, err := s.repo.ListStories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stories: %w", err)
	}
	return stories, nil
}

func (s *Service) Graph(ctx context.Context, storyID string) (Graph, error) {
	story, err := s.repo.GetStory(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("get story: %w", err)
	}
	scenes, err := s.repo.ListScenes(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("list scenes: %w", err)
	}
	edges, err := s.repo.ListEdges(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("list edges: %w", err)
	}
	return Graph{Story: story, Scenes: scenes, Edges: edges}, nil
}

func (s *Service) CreateScene(ctx context.Context, storyID, title, brief string, position Position) (Scene, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		return Scene{}, fmt.Errorf("get story: %w", err)
	}
	title, err := requiredText(title)
	if err != nil {
		return Scene{}, ErrInvalidTitle
	}
	now := s.now().UTC()
	scene := Scene{ID: s.newID(), StoryID: storyID, Title: title, Brief: strings.TrimSpace(brief), Position: position, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateScene(ctx, scene); err != nil {
		return Scene{}, fmt.Errorf("create scene: %w", err)
	}
	return scene, nil
}

func (s *Service) UpdateScene(ctx context.Context, sceneID string, title, brief *string, position *Position) (Scene, error) {
	return s.UpdateSceneAtRevision(ctx, sceneID, title, brief, position, nil)
}
func (s *Service) UpdateSceneAtRevision(ctx context.Context, sceneID string, title, brief *string, position *Position, expected *int64) (Scene, error) {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		return Scene{}, fmt.Errorf("get scene: %w", err)
	}
	if expected != nil && *expected != scene.Revision {
		return Scene{}, ErrConflict
	}
	if scene.AcceptedDraftID != "" && (title != nil || brief != nil) {
		return Scene{}, ErrSceneLocked
	}
	scene.Revision++
	if title != nil || brief != nil {
		scene.ContentRevision++
	}
	if title != nil {
		trimmed, err := requiredText(*title)
		if err != nil {
			return Scene{}, ErrInvalidTitle
		}
		scene.Title = trimmed
	}
	if brief != nil {
		scene.Brief = strings.TrimSpace(*brief)
	}
	if position != nil {
		scene.Position = *position
	}
	scene.UpdatedAt = s.now().UTC()
	if err := s.repo.UpdateScene(ctx, scene); err != nil {
		return Scene{}, fmt.Errorf("update scene: %w", err)
	}
	return scene, nil
}

func requiredText(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidTitle
	}
	return value, nil
}

func (s *Service) CreateEdge(ctx context.Context, storyID, fromID, toID string) (Edge, error) {
	if fromID == "" || toID == "" || fromID == toID {
		return Edge{}, ErrInvalidEdge
	}
	from, err := s.repo.GetScene(ctx, fromID)
	if err != nil {
		return Edge{}, fmt.Errorf("get source scene: %w", err)
	}
	to, err := s.repo.GetScene(ctx, toID)
	if err != nil {
		return Edge{}, fmt.Errorf("get target scene: %w", err)
	}
	if from.StoryID != storyID || to.StoryID != storyID {
		return Edge{}, ErrCrossStoryEdge
	}
	edges, err := s.repo.ListEdges(ctx, storyID)
	if err != nil {
		return Edge{}, fmt.Errorf("list edges: %w", err)
	}
	for _, edge := range edges {
		if edge.FromSceneID == fromID && edge.ToSceneID == toID {
			return Edge{}, ErrDuplicateEdge
		}
	}
	candidate := Edge{ID: s.newID(), StoryID: storyID, FromSceneID: fromID, ToSceneID: toID, CreatedAt: s.now().UTC()}
	if HasCycle(append(edges, candidate)) {
		return Edge{}, ErrCycle
	}
	if err := s.repo.CreateEdge(ctx, candidate); err != nil {
		return Edge{}, fmt.Errorf("create edge: %w", err)
	}
	return candidate, nil
}

func HasCycle(edges []Edge) bool {
	next := make(map[string][]string)
	nodes := make(map[string]struct{})
	for _, edge := range edges {
		next[edge.FromSceneID] = append(next[edge.FromSceneID], edge.ToSceneID)
		nodes[edge.FromSceneID] = struct{}{}
		nodes[edge.ToSceneID] = struct{}{}
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, child := range next[node] {
			if visit(child) {
				return true
			}
		}
		visiting[node] = false
		visited[node] = true
		return false
	}
	for node := range nodes {
		if visit(node) {
			return true
		}
	}
	return false
}
