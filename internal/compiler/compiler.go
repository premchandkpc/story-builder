package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/ledger"
)

type CompiledContext struct {
	CharacterCards []canon.Card            `json:"character_cards"`
	LocationCard   *canon.Card             `json:"location_card,omitempty"`
	BranchSummary  string                  `json:"branch_summary"`
	CharState      map[string]ledger.CharacterState `json:"char_state"`
	Lore           []string                `json:"lore"`
	BeatIntent     string                  `json:"beat_intent"`
	POV            string                  `json:"pov"`
	Tone           string                  `json:"tone"`
	TargetWords    int                     `json:"target_words"`
}

func (c CompiledContext) Hash() string {
	b, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Sprintf("compiler: failed to marshal context: %v", err))
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type CompileInput struct {
	StoryID       string
	NodeID        string
	CharacterRefs []string
	LocationRef   *string
	BranchSummary string
	BeatIntent    string
	POV           string
	Tone          string
	TargetWords   int
}

type CompilerService interface {
	Compile(input CompileInput) (*CompiledContext, error)
	VerifyBatch(ctxHash string, inputs []CompileInput) []string
}

type Generation struct {
	ID             string `json:"id"`
	NodeID         string `json:"node_id"`
	ContextHash    string `json:"context_hash"`
	PromptSnapshot string `json:"prompt_snapshot"`
	Output         string `json:"output"`
	Model          string `json:"model"`
	Accepted       bool   `json:"accepted"`
}

type GenerationService interface {
	Create(g *Generation) error
	Accept(id string) error
	Reject(id string) error
	GetByNode(nodeID string) ([]Generation, error)
	IsStale(nodeID string, currentHash string) (bool, error)
}
