package story

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalidTitle   = errors.New("title is required")
	ErrInvalidEdge    = errors.New("edge requires two different scenes")
	ErrCrossStoryEdge = errors.New("both scenes must belong to the story")
	ErrCycle          = errors.New("edge would create a cycle")
	ErrDuplicateEdge  = errors.New("edge already exists")
	ErrStaleDraft     = errors.New("draft is stale")
	ErrDraftNotReady  = errors.New("draft is not ready")
)
