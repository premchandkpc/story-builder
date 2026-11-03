package story

import "sort"

func ProjectCharacterStates(events []CanonEvent) []CharacterState {
	states := map[string]CharacterState{}
	sort.Slice(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	for _, event := range events {
		if event.Type != "character_state" {
			continue
		}
		state := states[event.CharacterID]
		state.StoryID = event.StoryID
		state.CharacterID = event.CharacterID
		state.AsOfCanonVersion = event.Seq
		state.Emotion = event.Emotion
		state.Knowledge = appendUnique(state.Knowledge, event.Learned)
		states[event.CharacterID] = state
	}
	out := make([]CharacterState, 0, len(states))
	for _, state := range states {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CharacterID < out[j].CharacterID })
	return out
}
func appendUnique(base, additional []string) []string {
	seen := map[string]bool{}
	for _, item := range base {
		seen[item] = true
	}
	for _, item := range additional {
		if item != "" && !seen[item] {
			base = append(base, item)
			seen[item] = true
		}
	}
	return base
}
