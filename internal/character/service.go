package character

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/event"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/memory"
)

type Service interface {
	CreateDefinition(ctx context.Context, name string, opts ...DefinitionOption) (*Definition, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (*Definition, error)
	UpdateDefinition(ctx context.Context, id uuid.UUID, opts ...DefinitionOption) (*Definition, error)
	ListDefinitions(ctx context.Context) ([]Definition, error)

	GetState(ctx context.Context, storyID, characterID, asOfScene uuid.UUID) (*State, error)
	GetStatesForScene(ctx context.Context, storyID, asOfScene uuid.UUID) ([]State, error)
	ApplyDelta(ctx context.Context, storyID, sceneID uuid.UUID, delta ledger.StateDelta) error

	StoreMemory(ctx context.Context, m *Memory) error
	SearchMemories(ctx context.Context, query RetrievalQuery) ([]RankedMemory, error)
	GetRecentMemories(ctx context.Context, storyID, characterID uuid.UUID, limit int) ([]Memory, error)
}

type DefinitionOption func(*Definition)

func WithPersona(p string) DefinitionOption {
	return func(d *Definition) { d.Persona = p }
}

func WithBackstory(b string) DefinitionOption {
	return func(d *Definition) { d.Backstory = b }
}

func WithMoralAlignment(m string) DefinitionOption {
	return func(d *Definition) { d.MoralAlignment = m }
}

func WithPersonality(p []string) DefinitionOption {
	return func(d *Definition) { d.Personality = p }
}

func WithFlaws(f []string) DefinitionOption {
	return func(d *Definition) { d.Flaws = f }
}

func WithGoals(g []string) DefinitionOption {
	return func(d *Definition) { d.Goals = g }
}

func WithTraits(t []string) DefinitionOption {
	return func(d *Definition) { d.Traits = t }
}

func WithVoiceSamples(v []string) DefinitionOption {
	return func(d *Definition) { d.VoiceSamples = v }
}

func WithRelationships(r map[string]string) DefinitionOption {
	return func(d *Definition) { d.Relationships = r }
}

type InMemoryService struct {
	defs  *canon.MemoryStore
	state *ledger.MemoryStore
	mem   *memory.MemoryStore
	mu    sync.RWMutex

	// local def cache for full Definition fields
	defCache map[uuid.UUID]*Definition
}

func NewInMemoryService() *InMemoryService {
	return &InMemoryService{
		defs:     canon.NewMemoryStore(),
		state:    ledger.NewMemoryStore(),
		mem:      memory.NewMemoryStore(),
		defCache: make(map[uuid.UUID]*Definition),
	}
}

func NewEventSourcedService(estore event.Store, bus event.Bus) *InMemoryService {
	s := &InMemoryService{
		defs:     canon.NewMemoryStore(),
		state:    ledger.NewEventSourcedStore(estore, bus),
		mem:      memory.NewMemoryStore(),
		defCache: make(map[uuid.UUID]*Definition),
	}
	s.state.ReplayOnStartup()
	return s
}

func (s *InMemoryService) CreateDefinition(ctx context.Context, name string, opts ...DefinitionOption) (*Definition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, err := s.defs.Create(name, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	d := &Definition{
		ID:            cc.ID,
		Version:       1,
		Name:          name,
		Persona:       cc.Persona,
		Backstory:     cc.Backstory,
		Traits:        cc.Traits,
		VoiceSamples:  cc.VoiceSamples,
		Relationships: cc.Relationships,
		CreatedAt:     cc.CreatedAt,
	}
	for _, opt := range opts {
		opt(d)
	}
	s.defCache[d.ID] = d
	return d, nil
}

func (s *InMemoryService) GetDefinition(ctx context.Context, id uuid.UUID) (*Definition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if d, ok := s.defCache[id]; ok {
		return d, nil
	}

	cc, err := s.defs.GetLatest(id)
	if err != nil {
		return nil, err
	}
	d := &Definition{
		ID:            cc.ID,
		Version:       cc.Version,
		Name:          cc.Name,
		Persona:       cc.Persona,
		Backstory:     cc.Backstory,
		MoralAlignment: cc.MoralAlignment,
		Personality:   cc.Personality,
		Flaws:         cc.Flaws,
		Goals:         cc.Goals,
		Traits:        cc.Traits,
		VoiceSamples:  cc.VoiceSamples,
		ParentID:      cc.ParentID,
		Relationships: cc.Relationships,
		CreatedAt:     cc.CreatedAt,
	}
	return d, nil
}

func (s *InMemoryService) UpdateDefinition(ctx context.Context, id uuid.UUID, opts ...DefinitionOption) (*Definition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, err := s.GetDefinition(ctx, id)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(d)
	}
	d.Version++

	_, err = s.defs.Update(id, d.Traits, d.VoiceSamples, d.Relationships)
	if err != nil {
		return nil, err
	}

	s.defCache[id] = d
	return d, nil
}

func (s *InMemoryService) ListDefinitions(ctx context.Context) ([]Definition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ccs, err := s.defs.List()
	if err != nil {
		return nil, err
	}
	result := make([]Definition, len(ccs))
	for i, cc := range ccs {
		result[i] = Definition{
			ID:            cc.ID,
			Version:       cc.Version,
			Name:          cc.Name,
			Persona:       cc.Persona,
			Backstory:     cc.Backstory,
			MoralAlignment: cc.MoralAlignment,
			Personality:   cc.Personality,
			Flaws:         cc.Flaws,
			Goals:         cc.Goals,
			Traits:        cc.Traits,
			VoiceSamples:  cc.VoiceSamples,
			ParentID:      cc.ParentID,
			Relationships: cc.Relationships,
			CreatedAt:     cc.CreatedAt,
		}
	}
	return result, nil
}

func (s *InMemoryService) GetState(ctx context.Context, storyID, characterID, asOfScene uuid.UUID) (*State, error) {
	cs, err := s.state.GetState(storyID, characterID, asOfScene)
	if err != nil {
		return nil, err
	}
	return stateFromLedger(cs), nil
}

func (s *InMemoryService) GetStatesForScene(ctx context.Context, storyID, asOfScene uuid.UUID) ([]State, error) {
	states, err := s.state.GetAllStates(storyID, asOfScene)
	if err != nil {
		return nil, err
	}
	result := make([]State, 0, len(states))
	for _, cs := range states {
		result = append(result, *stateFromLedger(cs))
	}
	return result, nil
}

func (s *InMemoryService) ApplyDelta(ctx context.Context, storyID, sceneID uuid.UUID, delta ledger.StateDelta) error {
	return s.state.ApplyDelta(storyID, sceneID, delta)
}

func (s *InMemoryService) StoreMemory(ctx context.Context, m *Memory) error {
	mm := &memory.Memory{
		StoryID:         m.StoryID,
		CharacterID:     m.CharacterID,
		SceneID:         m.SceneID,
		Type:            memory.MemType(m.Type),
		Summary:         m.Summary,
		Importance:      m.Importance,
		EmotionalWeight: m.EmotionalWeight,
		Embedding:       m.Embedding,
	}
	return s.mem.Create(mm)
}

func (s *InMemoryService) SearchMemories(ctx context.Context, query RetrievalQuery) ([]RankedMemory, error) {
	q := memory.RetrievalQuery{
		StoryID:     query.StoryID,
		CharacterID: query.CharacterID,
		SceneID:     query.SceneID,
		QueryText:   query.QueryText,
		MaxResults:  query.MaxResults,
		MinScore:    query.MinScore,
	}
	for _, t := range query.Types {
		q.Types = append(q.Types, memory.MemType(t))
	}

	ranked, err := s.mem.Search(q)
	if err != nil {
		return nil, err
	}
	result := make([]RankedMemory, len(ranked))
	for i, r := range ranked {
		result[i] = RankedMemory{
			Memory: Memory{
				ID:              r.ID,
				StoryID:         r.StoryID,
				CharacterID:     r.CharacterID,
				SceneID:         r.SceneID,
				Type:            MemoryType(r.Type),
				Summary:         r.Summary,
				Importance:      r.Importance,
				EmotionalWeight: r.EmotionalWeight,
				Embedding:       r.Embedding,
				CreatedAt:       r.CreatedAt,
			},
			Score: r.Score,
		}
	}
	return result, nil
}

func (s *InMemoryService) GetRecentMemories(ctx context.Context, storyID, characterID uuid.UUID, limit int) ([]Memory, error) {
	mems, err := s.mem.RetrieveRecent(storyID, characterID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]Memory, len(mems))
	for i, m := range mems {
		result[i] = Memory{
			ID:              m.ID,
			StoryID:         m.StoryID,
			CharacterID:     m.CharacterID,
			SceneID:         m.SceneID,
			Type:            MemoryType(m.Type),
			Summary:         m.Summary,
			Importance:      m.Importance,
			EmotionalWeight: m.EmotionalWeight,
			Embedding:       m.Embedding,
			CreatedAt:       m.CreatedAt,
		}
	}
	return result, nil
}

func stateFromLedger(cs *ledger.CharacterState) *State {
	return &State{
		StoryID:       cs.StoryID,
		CharacterID:   cs.CharacterID,
		AsOfScene:     cs.AsOfNode,
		Location:      cs.Location,
		Knows:         cs.Knows,
		DoesNotKnow:   cs.DoesNotKnow,
		Mood:          cs.Mood,
		Relationships: cs.Relationships,
		Items:         cs.Items,
		UpdatedAt:     cs.UpdatedAt,
	}
}

var _ Service = (*InMemoryService)(nil)

func Ensure(ctx context.Context) error {
	return nil
}
