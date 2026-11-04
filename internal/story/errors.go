package story

import "errors"

var (
	ErrConflict         = errors.New("item changed; reload and retry")
	ErrInvalidCharacter = errors.New("invalid character definition")
	ErrNotFound         = errors.New("not found")
	ErrInvalidTitle     = errors.New("title is required")
	ErrInvalidEdge      = errors.New("edge requires two different scenes")
	ErrCrossStoryEdge   = errors.New("both scenes must belong to the story")
	ErrCycle            = errors.New("edge would create a cycle")
	ErrDuplicateEdge    = errors.New("edge already exists")
	ErrStaleDraft       = errors.New("draft is stale")
	ErrDraftNotReady    = errors.New("draft is not ready")
	ErrSceneLocked      = errors.New("accepted scenes are locked")
	ErrParentsPending   = errors.New("all parent scenes must be accepted first")
	ErrDraftInProgress  = errors.New("a draft for this context is already generating")
)
