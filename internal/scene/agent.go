package scene

import "fmt"

func BuildAgentPrompt(input AgentPromptInput) (systemPrompt, userMessage string) {
	systemPrompt = fmt.Sprintf(`You are %s.%s

Current location: %s
Your mood: %s
What you know: %s
What you don't know: %s

Scene goal: %s

Respond in character. Use natural dialogue and action. Keep responses concise and driven by your character's personality.`,
		input.CharacterName,
		traitsBlock(input.Traits, input.VoiceSamples),
		input.Location,
		input.Mood,
		bulletList(input.Knows),
		bulletList(input.DoesNotKnow),
		input.BeatIntent,
	)

	userMessage = fmt.Sprintf(`Scene: %s

Previous events:
%s

%s's action:`,
		input.SituationFlow,
		input.PreviousTurns,
		input.CharacterName,
	)

	return
}

func traitsBlock(traits, voices []string) string {
	var s string
	if len(traits) > 0 {
		s += "\nTraits: " + joinComma(traits)
	}
	if len(voices) > 0 {
		s += "\nVoice: " + joinComma(voices)
	}
	return s
}

func bulletList(items []string) string {
	if len(items) == 0 {
		return "nothing yet"
	}
	s := "\n"
	for _, item := range items {
		s += "- " + item + "\n"
	}
	return s
}

func joinComma(items []string) string {
	s := ""
	for i, item := range items {
		if i > 0 {
			s += ", "
		}
		s += item
	}
	return s
}
