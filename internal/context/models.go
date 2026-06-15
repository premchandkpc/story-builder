package context

import (
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/memory"
	"github.com/premchand/story-builder/internal/relationship"
	"github.com/premchand/story-builder/internal/runtime"
)

type AssemblyInput struct {
	StoryID      uuid.UUID                         `json:"story_id"`
	ChapterID    uuid.UUID                         `json:"chapter_id,omitempty"`
	SceneID      uuid.UUID                         `json:"scene_id"`
	CharIDs      []uuid.UUID                       `json:"character_ids"`
	Runtime      *runtime.SceneRuntime             `json:"runtime,omitempty"`
	CharRuntimes map[uuid.UUID]*runtime.CharRuntime `json:"character_runtimes,omitempty"`
}

type AssembledContext struct {
	Input        AssemblyInput                             `json:"input"`
	Memories     []memory.RankedMemory                     `json:"memories"`
	Relationships map[uuid.UUID]map[uuid.UUID]*relationship.Relationship `json:"relationships"`
	Snapshot     *runtime.RuntimeSnapshot                   `json:"snapshot,omitempty"`
}

type AssemblyEngine struct {
	memStore  memory.Store
	relStore  relationship.Store
	rtStore   runtime.Store
}

func NewAssemblyEngine(memStore memory.Store, relStore relationship.Store, rtStore runtime.Store) *AssemblyEngine {
	return &AssemblyEngine{
		memStore:  memStore,
		relStore:  relStore,
		rtStore:   rtStore,
	}
}

func (e *AssemblyEngine) Assemble(input AssemblyInput) (*AssembledContext, error) {
	ctx := &AssembledContext{Input: input}

	mems, err := e.memStore.Search(memory.RetrievalQuery{
		StoryID:    input.StoryID,
		MaxResults: 20,
		MinScore:   0.1,
	})
	if err != nil {
		mems = nil
	}
	ctx.Memories = mems

	rels := make(map[uuid.UUID]map[uuid.UUID]*relationship.Relationship)
	for _, cID := range input.CharIDs {
		charRels, err := e.relStore.GetAllForCharacter(input.StoryID, cID)
		if err != nil {
			continue
		}
		relMap := make(map[uuid.UUID]*relationship.Relationship)
		for i := range charRels {
			r := charRels[i]
			otherID := r.CharA
			if otherID == cID {
				otherID = r.CharB
			}
			relMap[otherID] = &r
		}
		rels[cID] = relMap
	}
	ctx.Relationships = rels

	snap := &runtime.RuntimeSnapshot{
		StoryID:       input.StoryID,
		SceneID:       input.SceneID,
		Characters:    make(map[uuid.UUID]runtime.CharRuntime),
		Relationships: make(map[uuid.UUID]map[uuid.UUID]runtime.RelSnapshot),
	}
	if input.Runtime != nil {
		for cid, cr := range input.Runtime.CharStates {
			snap.Characters[cid] = cr
		}
		for cid, relMap := range input.Runtime.Relationships {
			snap.Relationships[cid] = relMap
		}
	}
	ctx.Snapshot = snap

	return ctx, nil
}
