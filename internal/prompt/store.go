package prompt

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/llm"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]*PromptTemplate
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]*PromptTemplate),
	}
}

func (s *MemoryStore) Save(tmpl *PromptTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmpl.ID = uuid.New()
	s.data[tmpl.Name] = tmpl
	return nil
}

func (s *MemoryStore) Get(name string) (*PromptTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tmpl, ok := s.data[name]
	if !ok {
		return nil, fmt.Errorf("prompt template %q not found", name)
	}
	return tmpl, nil
}

func (s *MemoryStore) List() ([]PromptTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PromptTemplate, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, *v)
	}
	return out, nil
}

func (s *MemoryStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, name)
	return nil
}

func DefaultTemplates() []*PromptTemplate {
	return []*PromptTemplate{
		{
			Name:        "scene_prose",
			Model:       llm.ModelSonnet,
			Temperature: 0.8,
			MaxTokens:   4096,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a fiction co-writer writing a narrative scene.", Priority: 1},
				{ID: LayerCulture, Strategy: MergeMerge, System: "", Priority: 2},
				{ID: LayerStory, Strategy: MergeOverride, System: "Canon is law. Never contradict established facts.", Priority: 3},
				{ID: LayerMemory, Strategy: MergeAppend, System: "", Priority: 4},
				{ID: LayerChapter, Strategy: MergeMerge, System: "", Priority: 5},
				{ID: LayerCharacter, Strategy: MergeMerge, System: "", Priority: 6},
				{ID: LayerScene, Strategy: MergeOverride, Template: "Write the scene as described below.", Priority: 7},
				{ID: LayerSafety, Strategy: MergeOverride, System: "HARD RULES:\n1. Canon is law.\n2. No new named characters/locations.\n3. Dialogue must match voice samples.\n4. Output prose only.", Priority: 8},
			},
		},
		{
			Name:        "state_extract",
			Model:       llm.ModelLocal,
			Temperature: 0.0,
			MaxTokens:   2048,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a continuity clerk. Extract state deltas from the scene.", Priority: 1},
				{ID: LayerScene, Strategy: MergeOverride, Template: "Extract character state changes from this scene.", Priority: 2},
			},
		},
		{
			Name:        "summary_update",
			Model:       llm.ModelLocal,
			Temperature: 0.2,
			MaxTokens:   1024,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You maintain a running plot summary for one storyline branch.", Priority: 1},
			},
		},
		{
			Name:        "canon_validate",
			Model:       llm.ModelHaiku,
			Temperature: 0.0,
			MaxTokens:   2048,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a strict continuity editor. Check draft against canon.", Priority: 1},
			},
		},
	}
}
