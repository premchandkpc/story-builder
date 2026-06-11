package ledger

import (
	"time"

	"github.com/google/uuid"
)

type CharacterState struct {
	StoryID     uuid.UUID `json:"story_id"`
	CharacterID uuid.UUID `json:"character_id"`
	AsOfNode    uuid.UUID `json:"as_of_node"`
	Location    string    `json:"location"`
	Knows       []string  `json:"knows"`
	DoesNotKnow []string  `json:"does_not_know"`
	Mood        string    `json:"mood"`
	Relationships map[string]string `json:"relationships"`
	Items       []string  `json:"items"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StateDelta struct {
	Character            uuid.UUID `json:"character"`
	NewLocation          string    `json:"new_location,omitempty"`
	Learned              []string  `json:"learned,omitempty"`
	Mood                 string    `json:"mood,omitempty"`
	RelationshipChanges  []RelationshipChange `json:"relationship_changes,omitempty"`
	ItemsGained          []string  `json:"items_gained,omitempty"`
	ItemsLost            []string  `json:"items_lost,omitempty"`
}

type RelationshipChange struct {
	With   uuid.UUID `json:"with"`
	Change string    `json:"change"`
}

type StateDeltas struct {
	Deltas      []StateDelta `json:"deltas"`
	OpenThreads []string     `json:"open_threads"`
}

type LedgerService interface {
	GetState(storyID, characterID, asOfNode uuid.UUID) (*CharacterState, error)
	GetAllStates(storyID, asOfNode uuid.UUID) (map[uuid.UUID]*CharacterState, error)
	ApplyDelta(storyID, nodeID uuid.UUID, delta StateDelta) error
	ApplyDeltas(storyID, nodeID uuid.UUID, deltas StateDeltas) error
	GetStateAtBranch(storyID, forkNode, branchNode uuid.UUID) (map[uuid.UUID]*CharacterState, error)
}

func DeriveDoesNotKnow(allFacts []string, knows []string) []string {
	known := make(map[string]struct{}, len(knows))
	for _, k := range knows {
		known[k] = struct{}{}
	}
	var unknown []string
	for _, f := range allFacts {
		if _, ok := known[f]; !ok {
			unknown = append(unknown, f)
		}
	}
	return unknown
}
