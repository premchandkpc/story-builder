package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
)

type CharacterAgentInstance struct {
	Character *domain.Character
	State     *CharacterAgentState
	Spec      AgentSpec

	eventCh chan CharacterEvent
	done    chan struct{}
	closeMu sync.Mutex
	closed  bool
}

type CharacterManager struct {
	mu       sync.RWMutex
	agents   map[string]*CharacterAgentInstance
	registry *AgentRegistry
	llmCli   llm.LLMClient
	proseSvc llm.ProseService
	eventBus events.Bus
}

func NewCharacterManager(registry *AgentRegistry, llmCli llm.LLMClient, proseSvc llm.ProseService, eventBus events.Bus) *CharacterManager {
	return &CharacterManager{
		agents:   make(map[string]*CharacterAgentInstance),
		registry: registry,
		llmCli:   llmCli,
		proseSvc: proseSvc,
		eventBus: eventBus,
	}
}

func (m *CharacterManager) StartAgent(ctx context.Context, c *domain.Character) (*CharacterAgentInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.agents[c.CharID]; ok {
		return nil, fmt.Errorf("character agent %s already running", c.CharID)
	}

	state := &CharacterAgentState{
		CharacterID: c.CharID,
		StoryID:     c.StoryID,
		Name:        c.Name,
		RelState:    make(map[string]*RelState),
	}

	spec := NewCharacterAgentSpec(c.CharID, m.llmCli, m.proseSvc, state)

	inst := &CharacterAgentInstance{
		Character: c,
		State:     state,
		Spec:      spec,
		eventCh:   make(chan CharacterEvent, 64),
		done:      make(chan struct{}),
	}

	m.agents[c.CharID] = inst

	m.registry.Register(spec)

	go m.agentLoop(inst)

	slog.Info("character manager: started agent", "charId", c.CharID, "name", c.Name)
	return inst, nil
}

func (m *CharacterManager) StopAgent(charID string) {
	m.mu.Lock()
	inst, ok := m.agents[charID]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.agents, charID)
	m.mu.Unlock()

	inst.closeMu.Lock()
	if !inst.closed {
		inst.closed = true
		close(inst.done)
	}
	inst.closeMu.Unlock()

	slog.Info("character manager: stopped agent", "charId", charID)
}

func (m *CharacterManager) StopAll() {
	m.mu.Lock()
	agents := make([]*CharacterAgentInstance, 0, len(m.agents))
	for _, inst := range m.agents {
		agents = append(agents, inst)
	}
	m.mu.Unlock()

	for _, inst := range agents {
		inst.closeMu.Lock()
		if !inst.closed {
			inst.closed = true
			close(inst.done)
		}
		inst.closeMu.Unlock()
	}

	m.mu.Lock()
	m.agents = make(map[string]*CharacterAgentInstance)
	m.mu.Unlock()

	slog.Info("character manager: stopped all agents")
}

func (m *CharacterManager) GetAgent(charID string) *CharacterAgentInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[charID]
}

func (m *CharacterManager) BroadcastEvent(evt CharacterEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inst := range m.agents {
		select {
		case inst.eventCh <- evt:
		default:
			slog.Warn("character manager: agent event channel full, dropping",
				"charId", inst.Character.CharID, "eventType", evt.Type)
		}
	}
}

func (m *CharacterManager) QueryProposals(ctx context.Context) []CharacterProposal {
	m.mu.RLock()
	agents := make([]*CharacterAgentInstance, 0, len(m.agents))
	for _, inst := range m.agents {
		agents = append(agents, inst)
	}
	m.mu.RUnlock()

	var proposals []CharacterProposal
	for _, inst := range agents {
		spec := inst.Spec
		if spec.Runner == nil {
			continue
		}

		out, err := spec.Runner(ctx, AgentInput{
			Directive: "propose",
			Payload:   map[string]any{"context": "autonomous"},
		})
		if err != nil {
			slog.Debug("character manager: proposal failed",
				"charId", inst.Character.CharID, "error", err)
			continue
		}
		if out == nil {
			continue
		}

		inst.State.RecordAction("", 0, "propose", out.Content)

		proposals = append(proposals, CharacterProposal{
			CharacterID: inst.Character.CharID,
			ActionType:  "dialogue",
			Content:     out.Content,
			Priority:    5,
		})
	}

	return proposals
}

func (m *CharacterManager) agentLoop(inst *CharacterAgentInstance) {
	for {
		select {
		case <-inst.done:
			return
		case evt := <-inst.eventCh:
			m.processEvent(inst, evt)
		}
	}
}

func (m *CharacterManager) processEvent(inst *CharacterAgentInstance, evt CharacterEvent) {
	switch evt.Type {
	case EventSceneStart:
		initAgentState(inst, evt)
	case EventTurnComplete:
		updateFromTurn(inst, evt)
	case EventSceneEnd:
		slog.Debug("character manager: scene end for agent",
			"charId", inst.Character.CharID)
	case EventStateUpdate:
		if data, ok := evt.Data["emotion"].(string); ok {
			inst.State.Lock()
			inst.State.CurrentEmotion = data
			inst.State.Unlock()
		}
	}
}

func initAgentState(inst *CharacterAgentInstance, evt CharacterEvent) {
	inst.State.Lock()
	defer inst.State.Unlock()

	if data, ok := evt.Data["emotion"].(string); ok && data != "" {
		inst.State.CurrentEmotion = data
	}
	if data, ok := evt.Data["mood"].(string); ok && data != "" {
		inst.State.CurrentMood = data
	}
	if data, ok := evt.Data["goal"].(string); ok && data != "" {
		inst.State.ActiveGoal = data
	}
	if data, ok := evt.Data["knowledge"].([]string); ok {
		inst.State.Knowledge = data
	}
	if data, ok := evt.Data["doesNotKnow"].([]string); ok {
		inst.State.KnowledgeGaps = data
	}

	for _, goal := range inst.Character.Goals {
		if inst.State.ActiveGoal == "" {
			inst.State.ActiveGoal = goal
			inst.State.Plan = &ActionPlan{
				Goal:     goal,
				Steps:    []string{},
				Priority: 5,
				Active:   true,
				FormedAt: time.Now(),
			}
			break
		}
	}
}

func updateFromTurn(inst *CharacterAgentInstance, evt CharacterEvent) {
	if data, ok := evt.Data["emotion"].(string); ok && data != "" {
		inst.State.Lock()
		inst.State.CurrentEmotion = data
		inst.State.Unlock()
	}
	if data, ok := evt.Data["content"].(string); ok && data != "" {
		inst.State.RecordDialogue(data)
	}
}

func (m *CharacterManager) SnapshotState(charID string) *AgentStateSnapshot {
	m.mu.RLock()
	inst, ok := m.agents[charID]
	m.mu.RUnlock()
	if !ok {
		return nil
	}

	inst.State.mu.RLock()
	defer inst.State.mu.RUnlock()

	snap := &AgentStateSnapshot{
		CharacterID:    inst.State.CharacterID,
		Name:           inst.State.Name,
		CurrentEmotion: inst.State.CurrentEmotion,
		CurrentMood:    inst.State.CurrentMood,
		ActiveGoal:     inst.State.ActiveGoal,
		SubGoals:       copySlice(inst.State.SubGoals),
		Knowledge:      copySlice(inst.State.Knowledge),
		KnowledgeGaps:  copySlice(inst.State.KnowledgeGaps),
		RecentDialogue: copySlice(inst.State.RecentDialogue),
		Running:        true,
	}

	for _, t := range inst.State.InternalThoughts {
		snap.InternalThoughts = append(snap.InternalThoughts, ThoughtSnapshot{
			Timestamp: t.Timestamp.Format(time.RFC3339),
			Thought:   t.Thought,
			Type:      t.Type,
		})
	}

	for _, a := range inst.State.RecentActions {
		snap.RecentActions = append(snap.RecentActions, ActionSnapshot{
			SceneID:    a.SceneID,
			ActionType: a.ActionType,
			Content:    a.Content,
		})
	}

	if inst.State.Plan != nil {
		snap.Plan = &PlanSnapshot{
			Goal:     inst.State.Plan.Goal,
			Steps:    copySlice(inst.State.Plan.Steps),
			Priority: inst.State.Plan.Priority,
			Active:   inst.State.Plan.Active,
		}
	}

	return snap
}

func (m *CharacterManager) SnapshotProposals(ctx context.Context) []ProposalSnapshot {
	props := m.QueryProposals(ctx)
	snaps := make([]ProposalSnapshot, 0, len(props))
	for _, p := range props {
		snaps = append(snaps, ProposalSnapshot{
			CharacterID: p.CharacterID,
			ActionType:  p.ActionType,
			Content:     p.Content,
			Priority:    p.Priority,
		})
	}
	return snaps
}

func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

func EnsureAgentsRunning(ctx context.Context, mgr *CharacterManager, characters []*domain.Character) {
	for _, c := range characters {
		if existing := mgr.GetAgent(c.CharID); existing != nil {
			continue
		}
		_, err := mgr.StartAgent(ctx, c)
		if err != nil {
			slog.Warn("character manager: failed to start agent",
				"charId", c.CharID, "error", err)
		}
	}
}
