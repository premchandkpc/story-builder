package memory

import (
	"github.com/premchand/story-builder/internal/story"
	"slices"
)

// Repository callers and providers must not share mutable slices with stored
// definitions or historical generation snapshots.
func cloneCharacter(value story.Character) story.Character {
	value.Traits = slices.Clone(value.Traits)
	value.Goals = slices.Clone(value.Goals)
	value.Fears = slices.Clone(value.Fears)
	value.Flaws = slices.Clone(value.Flaws)
	value.Values = slices.Clone(value.Values)
	value.Constraints = slices.Clone(value.Constraints)
	value.Secrets = slices.Clone(value.Secrets)
	value.InitialKnowledge = slices.Clone(value.InitialKnowledge)
	value.Relationships = slices.Clone(value.Relationships)
	return value
}
func cloneDraft(value story.Draft) story.Draft {
	value.ProposedEvents = slices.Clone(value.ProposedEvents)
	for i := range value.ProposedEvents {
		value.ProposedEvents[i].Learned = slices.Clone(value.ProposedEvents[i].Learned)
	}
	if value.Input != nil {
		input := *value.Input
		input.Parents = slices.Clone(input.Parents)
		input.Characters = slices.Clone(input.Characters)
		input.States = slices.Clone(input.States)
		for i := range input.Characters {
			input.Characters[i] = cloneCharacter(input.Characters[i])
		}
		for i := range input.States {
			input.States[i].Knowledge = slices.Clone(input.States[i].Knowledge)
		}
		value.Input = &input
	}
	return value
}
