package llm

import (
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/ledger"
)

type ModelTier string

const (
	ModelSonnet ModelTier = "claude-sonnet"
	ModelHaiku  ModelTier = "claude-haiku"
	ModelLocal  ModelTier = "local-7b"
)

type PromptParams struct {
	CharacterCards []canon.Card
	LocationCard   *canon.Card
	Lore           []string
	CharState      map[string]interface{}
	BranchSummary  string
	BeatIntent     string
	POV            string
	Tone           string
	TargetWords    int
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type CompletionRequest struct {
	Model       ModelTier
	System      string
	UserMessage string
	Temperature float64
	MaxTokens   int
	Tools       []ToolDefinition
	ToolChoice  string
	MaxRetries  int
}

type CompletionResponse struct {
	Content string
	ToolUse map[string]interface{}
	Model   string
}

type ProseService interface {
	GenerateScene(params PromptParams) (*CompletionResponse, error)
}

type ExtractionService interface {
	ExtractState(sceneText string, roster map[string]string) (*ledger.StateDeltas, error)
}

type SummaryService interface {
	UpdateSummary(previousSummary, newScene string) (string, error)
}

type MergeService interface {
	MergeBranches(summaryA, summaryB, timelineNote string) (map[string]interface{}, error)
}

type ValidationService interface {
	ValidateAgainstCanon(canonXML, charState, draft string) (map[string]interface{}, error)
}

type OutlineService interface {
	GenerateOutline(synopsis string) (*StoryOutline, error)
}

type LLMClient interface {
	Complete(req CompletionRequest) (*CompletionResponse, error)
}

type PromptTemplate string

const (
	PromptSceneProse    PromptTemplate = "scene_prose"
	PromptStateExtract  PromptTemplate = "state_extract"
	PromptSummaryUpdate PromptTemplate = "summary_update"
	PromptJoinMerge     PromptTemplate = "join_merge"
	PromptCanonValidate PromptTemplate = "canon_validate"
	PromptOutlineStory  PromptTemplate = "outline_story"
)

type StoryOutlineCharacter struct {
	Name           string   `json:"name"`
	Persona        string   `json:"persona"`
	Backstory      string   `json:"backstory"`
	MoralAlignment string   `json:"moral_alignment"`
	Personality    []string `json:"personality"`
	Flaws          []string `json:"flaws"`
	Goals          []string `json:"goals"`
	VoiceSamples   []string `json:"voice_samples,omitempty"`
}

type StoryOutlineBeat struct {
	Title          string   `json:"title"`
	BeatIntent     string   `json:"beat_intent"`
	CharacterNames []string `json:"character_names"`
	LocationName   string   `json:"location_name,omitempty"`
	POV            string   `json:"pov"`
	Tone           string   `json:"tone"`
	TargetWords    int      `json:"target_words"`
	Act            int      `json:"act"`
}

type StoryOutlineEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type StoryOutline struct {
	Title      string                  `json:"title"`
	Synopsis   string                  `json:"synopsis"`
	Characters []StoryOutlineCharacter `json:"characters"`
	Beats      []StoryOutlineBeat      `json:"beats"`
	Edges      []StoryOutlineEdge      `json:"edges"`
}

type PromptConfig struct {
	Template    PromptTemplate
	Model       ModelTier
	Temperature float64
	SystemText  string
}

var PromptRegistry = map[PromptTemplate]PromptConfig{
	PromptSceneProse: {
		Template:    PromptSceneProse,
		Model:       ModelSonnet,
		Temperature: 0.8,
		SystemText:  "You are a fiction co-writer. Write ONE scene and nothing else.",
	},
	PromptStateExtract: {
		Template:    PromptStateExtract,
		Model:       ModelLocal,
		Temperature: 0,
		SystemText:  "You are a continuity clerk. Read the scene and call record_state_deltas.",
	},
	PromptSummaryUpdate: {
		Template:    PromptSummaryUpdate,
		Model:       ModelLocal,
		Temperature: 0.2,
		SystemText:  "You maintain a running plot summary for one storyline branch.",
	},
	PromptJoinMerge: {
		Template:    PromptJoinMerge,
		Model:       ModelHaiku,
		Temperature: 0.2,
		SystemText:  "Two parallel storylines are converging. Merge their summaries.",
	},
	PromptCanonValidate: {
		Template:    PromptCanonValidate,
		Model:       ModelHaiku,
		Temperature: 0,
		SystemText:  "You are a strict continuity editor. Check this draft against canon.",
	},
	PromptOutlineStory: {
		Template:    PromptOutlineStory,
		Model:       ModelSonnet,
		Temperature: 0.7,
		SystemText:  "You are a master story architect. Given a synopsis, generate a structured story outline with characters, plot beats, and narrative flow.",
	},
}
