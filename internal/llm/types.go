package llm

import (
	"context"
)

type ModelTier string

const (
	ModelSonnet ModelTier = "claude-sonnet"
	ModelHaiku  ModelTier = "claude-haiku"
	ModelLocal  ModelTier = "local-7b"
)

type PromptParams struct {
	CharacterCards []CharacterCard
	LocationCard   *CharacterCard
	Lore           []string
	CharState      map[string]interface{}
	BranchSummary  string
	BeatIntent     string
	POV            string
	Tone           string
	TargetWords    int
	Memories       map[string][]string
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type CompletionRequest struct {
	Model        ModelTier
	System       string
	UserMessage  string
	Temperature  float64
	MaxTokens    int
	Tools        []ToolDefinition
	ToolChoice   string
	MaxRetries   int
	ValidateJSON bool
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CompletionResponse struct {
	Content string
	ToolUse map[string]interface{}
	Model   string
	Usage   Usage
}

type ProseService interface {
	GenerateScene(ctx context.Context, params PromptParams) (*CompletionResponse, error)
}

type ExtractionService interface {
	ExtractState(ctx context.Context, sceneText string, roster map[string]string) (*StateDeltas, error)
}

type SummaryService interface {
	UpdateSummary(ctx context.Context, previousSummary, newScene string) (string, error)
}

type MergeService interface {
	MergeBranches(ctx context.Context, summaryA, summaryB, timelineNote string) (map[string]interface{}, error)
}

type ValidationService interface {
	ValidateAgainstCanon(ctx context.Context, canonXML, charState, draft string) (map[string]interface{}, error)
}

type OutlineService interface {
	GenerateOutline(ctx context.Context, synopsis string) (*StoryOutline, error)
}

type TitleService interface {
	GenerateTitle(ctx context.Context, synopsis string) (string, error)
}

type LLMClient interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type PromptTemplate string

const (
	PromptSceneProse     PromptTemplate = "scene_prose"
	PromptStateExtract   PromptTemplate = "state_extract"
	PromptSummaryUpdate  PromptTemplate = "summary_update"
	PromptJoinMerge      PromptTemplate = "join_merge"
	PromptCanonValidate  PromptTemplate = "canon_validate"
	PromptOutlineStory   PromptTemplate = "outline_story"
	PromptGenerateTitle  PromptTemplate = "generate_title"
	PromptGenerateBible PromptTemplate = "generate_bible"
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
		Model:       ModelLocal,
		Temperature: 0.7,
		SystemText:  "You are a master story architect. Given a synopsis, generate a structured story outline with characters, plot beats, and narrative flow.",
	},
	PromptGenerateTitle: {
		Template:    PromptGenerateTitle,
		Model:       ModelLocal,
		Temperature: 0.5,
		SystemText:  "You are a creative title generator. Given a synopsis, generate a short, engaging story title (3-8 words). Return ONLY the title, no quotes or punctuation.",
	},
	PromptGenerateBible: {
		Template:    PromptGenerateBible,
		Model:       ModelSonnet,
		Temperature: 0.3,
		SystemText:  "You are a world-building expert. Given a story synopsis, outline, and characters, generate a complete Story Bible as JSON. Cover world, dimensions, rules, magic, factions, cultures, tone, and narrative voice. Output valid JSON only — no markdown, no code fences.",
	},
}
