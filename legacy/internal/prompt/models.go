package prompt

import "github.com/google/uuid"

type LayerID string

const (
	LayerGlobal    LayerID = "global"
	LayerStory     LayerID = "story"
	LayerChapter   LayerID = "chapter"
	LayerScenario  LayerID = "scenario"
	LayerScene     LayerID = "scene"
	LayerFrame     LayerID = "frame"
	LayerCharacter LayerID = "character"
	LayerCulture   LayerID = "culture"
	LayerSafety    LayerID = "safety"
	LayerMemory    LayerID = "memory"
)

type MergeStrategy string

const (
	MergeOverride MergeStrategy = "override"
	MergeMerge    MergeStrategy = "merge"
	MergeAppend   MergeStrategy = "append"
	MergeReplace  MergeStrategy = "replace"
	MergeDisable  MergeStrategy = "disable"
)

type PromptLayer struct {
	ID        LayerID       `json:"id"`
	Strategy  MergeStrategy `json:"strategy"`
	System    string        `json:"system"`
	Template  string        `json:"template,omitempty"`
	Model     string        `json:"model,omitempty"`
	Priority  int           `json:"priority"`
	Version   int           `json:"version"`
}

type PromptTemplate struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Layers      []PromptLayer `json:"layers"`
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Version     int           `json:"version"`
	CreatedAt   string        `json:"created_at"`
}

type CompileRequest struct {
	StoryID     uuid.UUID
	ChapterID   uuid.UUID
	SceneID     uuid.UUID
	CharacterID uuid.UUID

	// Prompt text overrides — ScenePrompt becomes the user message
	StoryPrompt     string
	ChapterPrompt   string
	ScenePrompt     string
	CharacterPrompt string
	CulturePrompt   string
	MemoryContext   string

	// Dynamic context injected into system prompt via buildDynamicContext
	CanonXML      string
	CharStateXML  string
	BranchSummary string
	TargetWords   int

	// Pipeline inputs for extraction/validation/outline
	RosterJSON    string
	SceneText     string
	CompiledCanon string
	Synopsis      string
}

type CompiledPrompt struct {
	System        string
	User          string
	Model         string
	Temperature   float64
	MaxTokens     int
	LayersApplied []LayerID
}

type Store interface {
	Save(tmpl *PromptTemplate) error
	Get(name string) (*PromptTemplate, error)
	List() ([]PromptTemplate, error)
	Delete(name string) error
}
