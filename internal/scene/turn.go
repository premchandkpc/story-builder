package scene

import (
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
)

func WhoActsNext(ss graph.SceneStructure, previousTurns []SceneTurn) []uuid.UUID {
	if len(ss.CharacterOrder) == 0 {
		return nil
	}

	switch ss.FlowType {
	case graph.FlowMonologue:
		return ss.CharacterOrder[:1]

	case graph.FlowDialogue:
		if len(ss.CharacterOrder) < 2 {
			return ss.CharacterOrder[:1]
		}
		next := len(previousTurns) % len(ss.CharacterOrder)
		return ss.CharacterOrder[next : next+1]

	case graph.FlowRoundRobin:
		next := len(previousTurns) % len(ss.CharacterOrder)
		return ss.CharacterOrder[next : next+1]

	case graph.FlowParallel:
		return ss.CharacterOrder

	default:
		if len(previousTurns) == 0 {
			return ss.CharacterOrder[:1]
		}
		last := previousTurns[len(previousTurns)-1]
		lastIdx := -1
		for i, c := range ss.CharacterOrder {
			if len(last.ActorIDs) > 0 && c == last.ActorIDs[0] {
				lastIdx = i
				break
			}
		}
		next := (lastIdx + 1) % len(ss.CharacterOrder)
		return ss.CharacterOrder[next : next+1]
	}
}
