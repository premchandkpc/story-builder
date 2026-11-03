package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/trace"
)

const maxConcurrentAgents = 5

type CharManager interface {
	BroadcastEvent(evt CharacterEvent)
	QueryProposals(ctx context.Context) []CharacterProposal
}

type OrchestratorConfig struct {
	Registry      *AgentRegistry
	LLMClient     llm.LLMClient
	EventBus      events.Bus
	Timeouts      map[string]time.Duration
	CharManager   CharManager
	BudgetChecker TokenBudgetChecker
}

type Orchestrator struct {
	registry      *AgentRegistry
	llm           llm.LLMClient
	eventBus      events.Bus
	timeouts      map[string]time.Duration
	charManager   CharManager
	budgetChecker TokenBudgetChecker
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
		registry:      cfg.Registry,
		llm:           cfg.LLMClient,
		eventBus:      cfg.EventBus,
		timeouts:      timeouts,
		charManager:   cfg.CharManager,
		budgetChecker: cfg.BudgetChecker,
	}
}

func (o *Orchestrator) Plan(ctx context.Context, scene *domain.Scene) (*OrchestrationPlan, error) {
	slog.Info("orchestrator: planning scene turns", "sceneId", scene.ID, "flowType", scene.FlowType)
	_, planSpan := trace.StartSpan(ctx, "orchestrator.Plan")
	if planSpan != nil {
		trace.SetAttribute(planSpan, "sceneId", scene.ID)
		trace.SetAttribute(planSpan, "flowType", scene.FlowType)
	}
	defer trace.End(planSpan)

	var proposals []CharacterProposal
	if o.charManager != nil {
		proposals = o.charManager.QueryProposals(ctx)
		if len(proposals) > 0 {
			slog.Info("orchestrator: collected character proposals",
				"count", len(proposals))
			for _, p := range proposals {
				slog.Debug("orchestrator: proposal", "charId", p.CharacterID, "action", p.ActionType)
			}
		}
	}

	plan := &OrchestrationPlan{
		SceneID:   scene.ID,
		MaxTurns:  scene.MaxTurns,
		Proposals: proposals,
	}

	charAgentIDs := gatherCharacterAgentIDs(scene, proposals)

	switch scene.FlowType {
	case domain.FlowTypeMonologue:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "perform", Required: true, Blocking: false})
		}
		plan.TurnOrder = append(plan.TurnOrder,
			TurnStep{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		)
	case domain.FlowTypeDialogue:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "perform", Required: true, Blocking: false})
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "respond", Required: true, Blocking: false})
		}
		plan.TurnOrder = append(plan.TurnOrder,
			TurnStep{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		)
	case domain.FlowTypeRoundRobin:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "perform", Required: true, Blocking: false})
		}
		plan.TurnOrder = append(plan.TurnOrder,
			TurnStep{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeCanonGuard, Phase: "validate-step", Required: false, Blocking: true},
		)
	case domain.FlowTypeAction:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "act", Required: true, Blocking: false})
		}
		plan.TurnOrder = append(plan.TurnOrder,
			TurnStep{AgentType: domain.AgentTypeNarrator, Phase: "describe", Required: true, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeEditor, Phase: "pace", Required: false, Blocking: true},
		)
	case domain.FlowTypeSilent:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
			{AgentType: domain.AgentTypeNarrator, Phase: "describe", Required: true, Blocking: true},
		}
	default:
		plan.TurnOrder = []TurnStep{
			{AgentType: domain.AgentTypeDirector, Phase: "plan", Required: true, Blocking: true},
		}
		for _, cid := range charAgentIDs {
			plan.TurnOrder = append(plan.TurnOrder, TurnStep{AgentType: cid, Phase: "perform", Required: true, Blocking: false})
		}
		plan.TurnOrder = append(plan.TurnOrder,
			TurnStep{AgentType: domain.AgentTypeNarrator, Phase: "narrate", Required: true, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeEditor, Phase: "refine", Required: false, Blocking: true},
			TurnStep{AgentType: domain.AgentTypeCanonGuard, Phase: "validate", Required: false, Blocking: true},
		)
	}
	return plan, nil
}

func gatherCharacterAgentIDs(scene *domain.Scene, proposals []CharacterProposal) []string {
	seen := map[string]bool{}
	var ids []string

	for _, pid := range scene.Participants {
		if !seen[pid] {
			seen[pid] = true
			ids = append(ids, pid)
		}
	}

	for _, p := range proposals {
		if !seen[p.CharacterID] {
			seen[p.CharacterID] = true
			ids = append(ids, p.CharacterID)
		}
	}

	return ids
}

func (o *Orchestrator) Execute(ctx context.Context, plan *OrchestrationPlan, agentCtx *AgentContext, turnRepo SceneTurnRepository) (*OrchestrationResult, error) {
	result := &OrchestrationResult{SceneID: plan.SceneID}

	execCtx, execSpan := trace.StartSpan(ctx, "orchestrator.Execute")
	if execSpan != nil {
		trace.SetAttribute(execSpan, "sceneId", plan.SceneID)
		trace.SetAttribute(execSpan, "maxTurns", plan.MaxTurns)
	}
	defer trace.End(execSpan)

	if o.charManager != nil {
		o.charManager.BroadcastEvent(CharacterEvent{
			Type:      EventSceneStart,
			StoryID:   agentCtx.StoryID,
			SceneID:   plan.SceneID,
			Data: map[string]any{
				"scene_title": agentCtx.Scene.Title,
				"beat_intent": agentCtx.Scene.BeatIntent,
				"pov":         agentCtx.Scene.POV,
				"tone":        agentCtx.Scene.Tone,
			},
			Timestamp: time.Now(),
		})
	}

	var (
		turnNumber int
		turns      []*domain.SceneTurn
		turnsMu    sync.Mutex
	)
	sem := make(chan struct{}, maxConcurrentAgents)

	appendTurn := func(turn *domain.SceneTurn) {
		turnsMu.Lock()
		turns = append(turns, turn)
		turnsMu.Unlock()
	}

	// runBatch executes all steps in the batch concurrently (non-blocking)
	// or sequentially (blocking). It drains the pending parallel group and
	// serializes blocking steps as barriers.
	var runBatch func([]TurnStep) error
	runBatch = func(steps []TurnStep) error {
		// We batch non-blocking steps into wave groups separated by blocking
		// barriers. Each wave gets its own errgroup to avoid the errgroup.Wait()
		// cleanup cancellation poisoning subsequent blocking steps.
		var waveStart int
		for waveStart < len(steps) {
			// Find the end of this wave (up to but not including the next blocking step)
			waveEnd := waveStart
			for waveEnd < len(steps) && !steps[waveEnd].Blocking {
				waveEnd++
			}
			// Include the barrier step if any
			if waveEnd < len(steps) && steps[waveEnd].Blocking {
				waveEnd++
			}

			wave := steps[waveStart:waveEnd]
			waveStart = waveEnd

			// Separate the barrier (last blocking step, if present)
			var barrier *TurnStep
			if len(wave) > 0 && wave[len(wave)-1].Blocking {
				barrier = &wave[len(wave)-1]
				wave = wave[:len(wave)-1]
			}

			if len(wave) > 0 {
				g, gCtx := errgroup.WithContext(execCtx)
				for _, step := range wave {
					step := step
					turnNumber++
					n := turnNumber
					sem <- struct{}{}
					g.Go(func() error {
						defer func() { <-sem }()
						turn, err := o.runStep(gCtx, plan, agentCtx, turnRepo, step, n)
						if err != nil {
							if turn != nil {
								appendTurn(turn)
							}
							return err
						}
						if turn != nil {
							appendTurn(turn)
						}
						return nil
					})
				}
				if err := g.Wait(); err != nil {
					return err
				}
			}

			if barrier != nil {
				turnNumber++
				turn, err := o.runStep(execCtx, plan, agentCtx, turnRepo, *barrier, turnNumber)
				if err != nil {
					if turn != nil {
						appendTurn(turn)
					}
					return err
				}
				appendTurn(turn)
			}
		}

		return nil
	}

	if err := runBatch(plan.TurnOrder); err != nil {
		trace.SetError(execSpan, err)
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, err
	}

	result.Turns = turns

	if o.charManager != nil {
		o.charManager.BroadcastEvent(CharacterEvent{
			Type:    EventSceneEnd,
			StoryID: agentCtx.StoryID,
			SceneID: plan.SceneID,
			Data: map[string]any{
				"turn_count": len(result.Turns),
			},
			Timestamp: time.Now(),
		})
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

func (o *Orchestrator) runStep(ctx context.Context, plan *OrchestrationPlan, agentCtx *AgentContext, turnRepo SceneTurnRepository, step TurnStep, turnNumber int) (*domain.SceneTurn, error) {
	spec, ok := o.registry.Get(step.AgentType)
	if !ok {
		slog.Warn("orchestrator: agent not registered, skipping", "agentType", step.AgentType)
		if step.Required {
			return nil, fmt.Errorf("required agent %s not registered", step.AgentType)
		}
		return nil, nil
	}

	timeout := o.timeouts[step.AgentType]
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	turnCtx, turnSpan := trace.StartSpan(ctx, "turn."+step.AgentType+"."+step.Phase)
	turnCtx, cancel := context.WithTimeout(turnCtx, timeout)
	defer cancel()

	turn := &domain.SceneTurn{
		SceneID:   plan.SceneID,
		StoryID:   agentCtx.StoryID,
		Number:    turnNumber,
		AgentID:   spec.Name,
		Role:      spec.Role,
		Status:    domain.TurnStatusPending,
	}
	if turnRepo != nil {
		if err := turnRepo.Create(turnCtx, turn); err != nil {
			trace.SetError(turnSpan, err)
			return nil, fmt.Errorf("create turn: %w", err)
		}
	}
	if turnSpan != nil {
		trace.SetAttribute(turnSpan, "agentType", step.AgentType)
		trace.SetAttribute(turnSpan, "phase", step.Phase)
		trace.SetAttribute(turnSpan, "turnNumber", turnNumber)
		trace.SetAttribute(turnSpan, "turnId", turn.ID)
	}
	turnAgentCtx := *agentCtx
	turnAgentCtx.TurnID = turn.ID

	turn.Status = domain.TurnStatusRunning
	if turnRepo != nil {
		_ = turnRepo.Update(turnCtx, turn)
	}

	payload := map[string]any{"phase": step.Phase, "turnNumber": turnNumber}
	if step.AgentType == domain.AgentTypeDirector && len(plan.Proposals) > 0 {
		payload["proposals"] = plan.Proposals
	}

	if o.budgetChecker != nil {
		promptEst, compEst := agentBudgetEstimate(step.AgentType, spec.Model)
		if err := o.budgetChecker.CheckAndConsume(turnCtx, turnAgentCtx.StoryID, spec.Model, step.AgentType, promptEst, compEst); err != nil {
			turn.Status = domain.TurnStatusFailed
			turn.Error = err.Error()
			trace.SetError(turnSpan, err)
			if turnRepo != nil {
				_ = turnRepo.Update(ctx, turn)
			}
			trace.End(turnSpan)
			return nil, fmt.Errorf("budget exceeded for %s: %w", step.AgentType, err)
		}
	}

	start := time.Now()
	output, err := spec.Runner(turnCtx, AgentInput{
		Ctx:       &turnAgentCtx,
		Payload:   payload,
		Directive: step.Phase,
	})

	turn.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		turn.Status = domain.TurnStatusFailed
		turn.Error = err.Error()
		trace.SetError(turnSpan, err)
		if turnRepo != nil {
			_ = turnRepo.Update(ctx, turn)
		}
		trace.End(turnSpan)
		return turn, fmt.Errorf("required step %s failed: %w", step.AgentType, err)
	}

	turn.Status = domain.TurnStatusDone
	turn.Output = output.Content
	trace.End(turnSpan)
	if turnRepo != nil {
		_ = turnRepo.Update(ctx, turn)
	}

	if o.charManager != nil {
		evtType := EventTurnComplete
		isCharacter := spec.Role == "character"
		if isCharacter {
			evtType = EventCharAction
		}
		o.charManager.BroadcastEvent(CharacterEvent{
			Type:      evtType,
			StoryID:   agentCtx.StoryID,
			SceneID:   plan.SceneID,
			TurnID:    turn.ID,
			Data: map[string]any{
				"agentType": step.AgentType,
				"phase":     step.Phase,
				"content":   output.Content,
				"role":      spec.Role,
				"emotion":   extractEmotion(output),
			},
			Timestamp: time.Now(),
		})
	}

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

	return turn, nil
}

func agentBudgetEstimate(agentType, model string) (promptTokens, completionTokens int) {
	switch agentType {
	case domain.AgentTypeDirector:
		return 3000, 1500
	case domain.AgentTypeCharacter:
		return 1500, 500
	case domain.AgentTypeNarrator:
		return 2000, 1000
	case domain.AgentTypeEditor, domain.AgentTypeWorld, domain.AgentTypeArc:
		return 1500, 500
	case domain.AgentTypeCanonGuard, domain.AgentTypeCritic:
		return 1000, 300
	case domain.AgentTypeStateExtract, domain.AgentTypeMemory:
		return 1000, 500
	default:
		return 1000, 500
	}
}

func extractEmotion(output *AgentOutput) string {
	if output == nil {
		return ""
	}
	if e, ok := output.Decisions["emotion"].(string); ok {
		return e
	}
	return ""
}

func (o *Orchestrator) RunFinish(ctx context.Context, sceneID string, agentCtx *AgentContext, turnRepo SceneTurnRepository) error {
	slog.Info("orchestrator: scene finish phase", "sceneId", sceneID)

	fCtx, finishSpan := trace.StartSpan(ctx, "orchestrator.RunFinish")
	if finishSpan != nil {
		trace.SetAttribute(finishSpan, "sceneId", sceneID)
	}
	defer trace.End(finishSpan)

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
		stepCtx, stepSpan := trace.StartSpan(fCtx, "finish."+step.AgentType+"."+step.Phase)

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

		if stepSpan != nil {
			trace.SetAttribute(stepSpan, "agentType", step.AgentType)
			trace.SetAttribute(stepSpan, "phase", step.Phase)
		}

		timeout := o.timeouts[step.AgentType]
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		sCtx, cancel := context.WithTimeout(stepCtx, timeout)

		if o.budgetChecker != nil {
			promptEst, compEst := agentBudgetEstimate(step.AgentType, spec.Model)
			if err := o.budgetChecker.CheckAndConsume(sCtx, agentCtx.StoryID, spec.Model, step.AgentType, promptEst, compEst); err != nil {
				cancel()
				trace.SetError(stepSpan, err)
				trace.End(stepSpan)
				return fmt.Errorf("budget exceeded for %s: %w", step.AgentType, err)
			}
		}

		output, err := spec.Runner(sCtx, AgentInput{
			Ctx: agentCtx, Directive: step.Phase,
		})
		cancel()

		turn.Status = domain.TurnStatusDone
		if err != nil {
			turn.Status = domain.TurnStatusFailed
			turn.Error = err.Error()
			trace.SetError(stepSpan, err)
		} else {
			turn.Output = output.Content
		}
		if turnRepo != nil {
			_ = turnRepo.Update(ctx, turn)
		}
		trace.End(stepSpan)
	}
	return nil
}
