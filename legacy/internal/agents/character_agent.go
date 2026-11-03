package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/trace"
)

func NewCharacterAgentSpec(charID string, llmClient llm.LLMClient, proseSvc llm.ProseService, state *CharacterAgentState) AgentSpec {
	spec := AgentSpec{
		Name:     charID,
		Role:     "character",
		Model:    string(llm.ModelSonnet),
		MaxTurns: 5,
		SystemPrompt: `You are an autonomous Character agent in a narrative engine.

You role-play ONE specific character. Rules:
1. Stay in-character: voice, goals, beliefs, emotional state, secrets
2. Act on what your character knows — not what you as an author know
3. React to other characters' actions and scene pressure
4. Output dialogue, action, or internal response
5. Narrate only from your character's perspective

Your output should be first-person or third-person limited, showing what your character says and does. Use dialogue for speech, *asterisks for actions*, and (parentheticals for internal thoughts).

When asked to PROPOSE an action, suggest what your character wants to do next.
When asked to PERFORM/RESPOND/ACT, produce the actual in-character output.`,
		Runner: buildCharacterRunner(charID, llmClient, state),
	}
	return spec
}

func buildCharacterRunner(charID string, llmClient llm.LLMClient, state *CharacterAgentState) AgentRunner {
	return func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
		switch input.Directive {
		case "propose":
			return runCharacterProposal(ctx, charID, input, llmClient, state)
		default:
			return runCharacterTurn(ctx, charID, input, llmClient, state)
		}
	}
}

func runCharacterProposal(ctx context.Context, charID string, input AgentInput, llmClient llm.LLMClient, state *CharacterAgentState) (*AgentOutput, error) {
	ctx, span := trace.StartSpan(ctx, "agent.character."+charID+".propose")
	defer trace.End(span)

	character := findCharacter(input, charID)
	if character == nil {
		trace.SetAttribute(span, "skipped", true)
		return &AgentOutput{Status: "skip", Content: ""}, nil
	}

	trace.SetAttribute(span, "charId", charID)
	trace.SetAttribute(span, "charName", character.Name)

	sceneCtx := buildSceneContext(input)

	state.Lock()
	goalDesc := state.ActiveGoal
	emotion := state.CurrentEmotion
	plan := ""
	if state.Plan != nil && state.Plan.Active {
		plan = fmt.Sprintf("Current plan: %s (steps: %s)", state.Plan.Goal, strings.Join(state.Plan.Steps, ", "))
	}
	thoughts := ""
	if len(state.InternalThoughts) > 0 {
		last := state.InternalThoughts[len(state.InternalThoughts)-1]
		thoughts = fmt.Sprintf("Recent thought: %s", last.Thought)
	}
	state.Unlock()

	msg := fmt.Sprintf(`=== AUTONOMOUS PROPOSAL ===
%s

Character: %s (%s)
Active Goal: %s
Current Emotion: %s
%s
%s

Based on your character's goals, personality, and current state, what do you want to do RIGHT NOW in this scene?
Output a brief in-character statement of intention — what you say, do, or how you react.
Keep it 1-3 sentences. This is your character's initiative.`,
		sceneCtx, character.Name, character.Persona, goalDesc, emotion, plan, thoughts)

	if len(state.RecentDialogue) > 0 {
		msg += "\n\nRecent exchanges:\n" + strings.Join(state.RecentDialogue, "\n")
	}

	sysPrompt := fmt.Sprintf("You are %s. Propose what you want to do next, in-character.", character.Name)
	resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
		Model:       llm.ModelSonnet,
		System:      sysPrompt,
		UserMessage: msg,
		Temperature: 0.8,
		MaxTokens:   512,
	})
	if err != nil {
		trace.SetError(span, err)
		return nil, fmt.Errorf("character proposal llm: %w", err)
	}

	state.RecordThought(resp.Content, "proposal")

	return &AgentOutput{
		Content: resp.Content,
		Data:    map[string]any{"character": character, "charId": charID},
		Decisions: map[string]any{
			"character_id": charID,
			"proposal":     resp.Content,
			"emotion":      emotion,
		},
		Status: "proposal",
	}, nil
}

func runCharacterTurn(ctx context.Context, charID string, input AgentInput, llmClient llm.LLMClient, state *CharacterAgentState) (*AgentOutput, error) {
	ctx, span := trace.StartSpan(ctx, "agent.character."+charID+"."+input.Directive)
	defer trace.End(span)

	character := findCharacter(input, charID)
	if character == nil {
		err := fmt.Errorf("character %s not found in context", charID)
		trace.SetError(span, err)
		return nil, err
	}

	trace.SetAttribute(span, "charId", charID)
	trace.SetAttribute(span, "charName", character.Name)
	trace.SetAttribute(span, "directive", input.Directive)

	charState := findCharState(input, charID)

	state.Lock()
	internalThoughts := ""
	if len(state.InternalThoughts) > 0 {
		recent := state.InternalThoughts
		if len(recent) > 3 {
			recent = recent[len(recent)-3:]
		}
		var parts []string
		for _, t := range recent {
			parts = append(parts, fmt.Sprintf("[%s] %s", t.Type, t.Thought))
		}
		internalThoughts = "Internal thoughts:\n" + strings.Join(parts, "\n")
	}

	activeGoal := state.ActiveGoal
	if activeGoal == "" && len(character.Goals) > 0 {
		activeGoal = character.Goals[0]
	}

	recentActions := ""
	if len(state.RecentActions) > 0 {
		var parts []string
		for _, a := range state.RecentActions {
			parts = append(parts, fmt.Sprintf("%s", a.Content))
		}
		recentActions = "Recent actions:\n" + strings.Join(parts, "\n")
	}
	emotion := state.CurrentEmotion
	if emotion == "" && charState != nil && charState.EmotionalState != "" {
		emotion = charState.EmotionalState
	}
	state.Unlock()

	userMsg := buildCharacterTurnPrompt(input, character, charState, charID, activeGoal, emotion, internalThoughts, recentActions)

	resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
		Model:       llm.ModelSonnet,
		System:      fmt.Sprintf("You are %s. Respond in-character as %s.", character.Name, character.Name),
		UserMessage: userMsg,
		Temperature: 0.8,
		MaxTokens:   2048,
	})
	if err != nil {
		trace.SetError(span, err)
		return nil, fmt.Errorf("character agent llm: %w", err)
	}

	state.RecordDialogue(resp.Content)
	state.RecordAction(input.Ctx.SceneID, 0, input.Directive, resp.Content)

	updateStateFromTurn(state, resp.Content, input, charState)

	return &AgentOutput{
		Content: resp.Content,
		Data:    map[string]any{"character": character},
		Decisions: map[string]any{
			"character_id": charID,
			"emotion":      emotion,
			"action_type":  input.Directive,
		},
		Status: "success",
	}, nil
}

func updateStateFromTurn(state *CharacterAgentState, output string, input AgentInput, cs *domain.CharacterState) {
	state.Lock()
	defer state.Unlock()

	outputLower := strings.ToLower(output)
	if strings.Contains(outputLower, "angry") || strings.Contains(outputLower, "furious") {
		state.CurrentEmotion = "angry"
	} else if strings.Contains(outputLower, "sad") || strings.Contains(outputLower, "grief") {
		state.CurrentEmotion = "sad"
	} else if strings.Contains(outputLower, "happy") || strings.Contains(outputLower, "joy") {
		state.CurrentEmotion = "happy"
	} else if strings.Contains(outputLower, "fear") || strings.Contains(outputLower, "afraid") {
		state.CurrentEmotion = "afraid"
	} else if strings.Contains(outputLower, "surprise") || strings.Contains(outputLower, "shock") {
		state.CurrentEmotion = "surprised"
	} else if strings.Contains(outputLower, "confus") {
		state.CurrentEmotion = "confused"
	} else if strings.Contains(outputLower, "love") || strings.Contains(outputLower, "affection") {
		state.CurrentEmotion = "affectionate"
	}

	state.InternalThoughts = append(state.InternalThoughts, InternalThought{
		Timestamp: time.Now(),
		Thought:   fmt.Sprintf("After action: %s", output),
		Type:      "reflection",
	})
	if len(state.InternalThoughts) > 50 {
		state.InternalThoughts = state.InternalThoughts[len(state.InternalThoughts)-50:]
	}

	if cs != nil && cs.ActiveGoal != "" {
		state.ActiveGoal = cs.ActiveGoal
	}
}

func buildCharacterTurnPrompt(input AgentInput, character *domain.Character, charState *domain.CharacterState, charID, activeGoal, emotion, internalThoughts, recentActions string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Scene: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Beat Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Tone: %s | POV: %s\n\n", input.Ctx.Scene.Tone, input.Ctx.Scene.POV))

	b.WriteString(fmt.Sprintf("=== CHARACTER: %s ===\n", character.Name))
	b.WriteString(fmt.Sprintf("Persona: %s\n", character.Persona))
	if len(character.Traits) > 0 {
		b.WriteString(fmt.Sprintf("Traits: %s\n", strings.Join(character.Traits, ", ")))
	}
	if len(character.Goals) > 0 {
		b.WriteString(fmt.Sprintf("Goals: %s\n", strings.Join(character.Goals, "; ")))
	}
	if len(character.Flaws) > 0 {
		b.WriteString(fmt.Sprintf("Flaws: %s\n", strings.Join(character.Flaws, "; ")))
	}
	if character.Fear != "" {
		b.WriteString(fmt.Sprintf("Fear: %s\n", character.Fear))
	}
	if character.FalseBelief != "" {
		b.WriteString(fmt.Sprintf("False Belief: %s\n", character.FalseBelief))
	}
	if len(character.VoiceSamples) > 0 {
		b.WriteString(fmt.Sprintf("Voice: %s\n", strings.Join(character.VoiceSamples, " | ")))
	}

	b.WriteString(fmt.Sprintf("\nActive Goal: %s\n", activeGoal))
	if emotion != "" {
		b.WriteString(fmt.Sprintf("Emotion: %s\n", emotion))
	}
	if internalThoughts != "" {
		b.WriteString(internalThoughts + "\n")
	}
	if recentActions != "" {
		b.WriteString(recentActions + "\n")
	}

	if charState != nil {
		b.WriteString("\n=== CURRENT STATE ===\n")
		if charState.Mood != "" {
			b.WriteString(fmt.Sprintf("Mood: %s\n", charState.Mood))
		}
		if charState.PhysicalState != "" {
			b.WriteString(fmt.Sprintf("Physical: %s\n", charState.PhysicalState))
		}
		if charState.Location != "" {
			b.WriteString(fmt.Sprintf("Location: %s\n", charState.Location))
		}
		if len(charState.Knowledge) > 0 {
			b.WriteString(fmt.Sprintf("Knows: %s\n", strings.Join(charState.Knowledge, "; ")))
		}
		if len(charState.DoesNotKnow) > 0 {
			b.WriteString(fmt.Sprintf("Does NOT Know: %s\n", strings.Join(charState.DoesNotKnow, "; ")))
		}
	}

	if mems, ok := input.Ctx.Memories[charID]; ok && len(mems) > 0 {
		b.WriteString("\n=== RELEVANT MEMORIES ===\n")
		maxMem := 5
		if len(mems) < maxMem {
			maxMem = len(mems)
		}
		for _, m := range mems[:maxMem] {
			b.WriteString(fmt.Sprintf("- [%s] %s\n", m.Type, m.Content))
		}
	}

	if len(input.Ctx.Turns) > 0 {
		b.WriteString("\n=== RECENT TURNS ===\n")
		start := 0
		if len(input.Ctx.Turns) > 5 {
			start = len(input.Ctx.Turns) - 5
		}
		for _, t := range input.Ctx.Turns[start:] {
			b.WriteString(fmt.Sprintf("[%s]: %s\n", t.Role, t.Output))
		}
	}

	b.WriteString(fmt.Sprintf("\n=== YOUR TASK ===\n"))
	b.WriteString(fmt.Sprintf("Phase: %s\n", input.Directive))
	b.WriteString("Produce dialogue, action, or internal response for this character. Write in-character only.\n")

	return b.String()
}

func findCharacter(input AgentInput, charID string) *domain.Character {
	if input.Ctx == nil {
		return nil
	}
	for _, c := range input.Ctx.Characters {
		if c.CharID == charID || c.ID == charID {
			return c
		}
	}
	return nil
}

func findCharState(input AgentInput, charID string) *domain.CharacterState {
	if input.Ctx == nil {
		return nil
	}
	for _, s := range input.Ctx.CharStates {
		if s.CharacterID == charID {
			return s
		}
	}
	return nil
}

func buildSceneContext(input AgentInput) string {
	if input.Ctx == nil || input.Ctx.Scene == nil {
		return ""
	}
	s := input.Ctx.Scene
	ctx := fmt.Sprintf("Scene: %s\nBeat Intent: %s\nTone: %s | POV: %s\nFlow: %s\nLocation: %s",
		s.Title, s.BeatIntent, s.Tone, s.POV, s.FlowType, s.LocationRef)

	var participants []string
	for _, pid := range input.Ctx.ParticipantIDs {
		for _, c := range input.Ctx.Characters {
			if c.CharID == pid {
				participants = append(participants, c.Name)
				break
			}
		}
	}
	if len(participants) == 0 {
		for _, c := range input.Ctx.Characters {
			participants = append(participants, c.Name)
		}
	}
	if len(participants) > 0 {
		ctx += "\nPresent: " + strings.Join(participants, ", ")
	}
	return ctx
}
