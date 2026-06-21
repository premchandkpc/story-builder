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

type AgentRegistry struct {
	mu      sync.RWMutex
	agents  map[string]AgentSpec
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]AgentSpec),
	}
}

func (r *AgentRegistry) Register(spec AgentSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[spec.Name] = spec
}

func (r *AgentRegistry) Get(name string) (AgentSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.agents[name]
	return spec, ok
}

func (r *AgentRegistry) List() []AgentSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	specs := make([]AgentSpec, 0, len(r.agents))
	for _, spec := range r.agents {
		specs = append(specs, spec)
	}
	return specs
}

type OrchestratorConfig struct {
	Registry   *AgentRegistry
	LLMClient  llm.LLMClient
	EventBus   events.Bus
	Timeouts   map[string]time.Duration
}

type Orchestrator struct {
	registry  *AgentRegistry
	llm       llm.LLMClient
	eventBus  events.Bus
	timeouts  map[string]time.Duration
}

func NewOrchestrator(cfg OrchestratorConfig) *Orchestrator {
	timeouts := cfg.Timeouts
	if timeouts == nil {
		timeouts = map[string]time.Duration{
			domain.AgentTypeDirector:    30 * time.Second,
			domain.AgentTypeCharacter:   60 * time.Second,
			domain.AgentTypeNarrator:    30 * time.Second,
			domain.AgentTypeEditor:      20 * time.Second,
			domain.AgentTypeCanonGuard:  15 * time.Second,
			domain.AgentTypeCritic:      15 * time.Second,
			domain.AgentTypeStateExtract: 30 * time.Second,
			domain.AgentTypeWorld:       30 * time.Second,
			domain.AgentTypeArc:         20 * time.Second,
			domain.AgentTypeMemory:      30 * time.Second,
		}
	}
	return &Orchestrator{
		registry: cfg.Registry,
		llm:      cfg.LLMClient,
		eventBus: cfg.EventBus,
		timeouts: timeouts,
	}
}

func (o *Orchestrator) Plan(ctx context.Context, scene *domain.Scene) (*OrchestrationPlan, error) {
	slog.Info("orchestrator: planning scene turns", "sceneId", scene.ID, "flowType", scene.FlowType)

	plan := &OrchestrationPlan{
		SceneID:  scene.ID,
		MaxTurns: scene.MaxTurns,
	}

	switch scene.FlowType {
	case domain.FlowTypeMonologue:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCharacter, Phase: "perform", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		}
	case domain.FlowTypeDialogue:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCharacter, Phase: "perform", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeCharacter, Phase: "respond", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		}
	case domain.FlowTypeRoundRobin:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCharacter, Phase: "perform", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCanonGuard, Phase: "validate-step", Required: false, Blocking: true},
		}
	case domain.FlowTypeAction:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCharacter, Phase: "act", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "describe", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "pace", Required: false, Blocking: true},
		}
	case domain.FlowTypeSilent:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeNarrator, Phase: "describe", Required: true, Blocking: true},
		}
	default:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeCharacter, Phase: "perform", Required: true, Blocking: false},
			{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		}
	}
	return plan, nil
}

func (o *Orchestrator) Execute(ctx context.Context, plan *OrchestrationPlan, agentCtx *AgentContext, turnRepo SceneTurnRepository) (*OrchestrationResult, error) {
	result := &OrchestrationResult{SceneID: plan.SceneID}

	for i, step := range plan.TurnOrder {
		spec, ok := o.registry.Get(step.AgentType)
		if !ok {
			slog.Warn("orchestrator: agent not registered, skipping", "agentType", step.AgentType)
			if step.Required {
				return nil, fmt.Errorf("required agent %s not registered", step.AgentType)
			}
			continue
		}

		timeout := o.timeouts[step.AgentType]
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		turnCtx, cancel := context.WithTimeout(ctx, timeout)

		turn := &domain.SceneTurn{
			SceneID:   plan.SceneID,
			StoryID:   agentCtx.StoryID,
			Number:    i + 1,
			AgentID:   spec.Name,
			Role:      spec.Role,
			Status:    domain.TurnStatusPending,
		}
		if turnRepo != nil {
			if err := turnRepo.Create(turnCtx, turn); err != nil {
				cancel()
				return nil, fmt.Errorf("create turn: %w", err)
			}
		}
		agentCtx.TurnID = turn.ID

		turn.Status = domain.TurnStatusRunning
		if turnRepo != nil {
			_ = turnRepo.Update(turnCtx, turn)
		}

		start := time.Now()
		output, err := spec.Runner(turnCtx, AgentInput{
			Ctx:       agentCtx,
			Payload:   map[string]any{"phase": step.Phase, "turnNumber": i + 1},
			Directive: step.Phase,
		})
		cancel()

		turn.DurationMs = time.Since(start).Milliseconds()

		if err != nil {
			turn.Status = domain.TurnStatusFailed
			turn.Error = err.Error()
					errMsg := fmt.Errorf("required step %s failed: %w", step.AgentType, err)
			result.Error = errMsg.Error()
			if turnRepo != nil {
				_ = turnRepo.Update(ctx, turn)
			}
			return result, errMsg
		} else {
			turn.Status = domain.TurnStatusDone
			turn.Output = output.Content
		}

		if turnRepo != nil {
			_ = turnRepo.Update(ctx, turn)
		}
		result.Turns = append(result.Turns, turn)

		if o.eventBus != nil {
			_ = o.eventBus.Publish(ctx, events.Event{
				Type:    events.EventAgentTurnCompleted,
				StoryID: agentCtx.StoryID,
				SceneID: plan.SceneID,
				Data: map[string]any{
					"turnId":    turn.ID,
					"agentType": step.AgentType,
					"phase":     step.Phase,
					"status":    turn.Status,
				},
			})
		}
	}

	if critic, ok := o.registry.Get(domain.AgentTypeCritic); ok {
		criticCtx, cancel := context.WithTimeout(ctx, o.timeouts[domain.AgentTypeCritic])
		defer cancel()
		out, err := critic.Runner(criticCtx, AgentInput{Ctx: agentCtx, Directive: "score"})
		if err == nil && out != nil {
			if score, ok := out.Decisions["score"].(float64); ok {
				result.CriticScore = score
			}
		}
	}

	return result, nil
}

func (o *Orchestrator) RunFinish(ctx context.Context, sceneID string, agentCtx *AgentContext, turnRepo SceneTurnRepository) error {
	slog.Info("orchestrator: scene finish phase", "sceneId", sceneID)

	finishOrder := []TurnStep{
		{AgentType: domain.AgentTypeStateExtract, Phase: "extract", Required: false, Blocking: true},
		{AgentType: domain.AgentTypeWorld, Phase: "world-check", Required: false, Blocking: false},
		{AgentType: domain.AgentTypeArc, Phase: "arc-check", Required: false, Blocking: false},
		{AgentType: domain.AgentTypeMemory, Phase: "memory-analysis", Required: false, Blocking: false},
		{AgentType: domain.AgentTypeDirector, Phase: "evaluate", Required: true, Blocking: true},
	}

	for _, step := range finishOrder {
		spec, ok := o.registry.Get(step.AgentType)
		if !ok {
			continue
		}
		turn := &domain.SceneTurn{
			SceneID: sceneID,
			StoryID: agentCtx.StoryID,
			AgentID: spec.Name,
			Role:    spec.Role,
			Status:  domain.TurnStatusPending,
		}
		if turnRepo != nil {
			_ = turnRepo.Create(ctx, turn)
		}

		timeout := o.timeouts[step.AgentType]
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		sCtx, cancel := context.WithTimeout(ctx, timeout)
		output, err := spec.Runner(sCtx, AgentInput{
			Ctx: agentCtx, Directive: step.Phase,
		})
		cancel()

		turn.Status = domain.TurnStatusDone
		if err != nil {
			turn.Status = domain.TurnStatusFailed
			turn.Error = err.Error()
		} else {
			turn.Output = output.Content
		}
		if turnRepo != nil {
			_ = turnRepo.Update(ctx, turn)
		}
	}
	return nil
}
