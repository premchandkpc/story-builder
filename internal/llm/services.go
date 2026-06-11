package llm

import (
	"encoding/json"
	"fmt"

	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/ledger"
)

func NewProseService(client LLMClient) *ProseServiceImpl {
	return &ProseServiceImpl{client: client}
}

type ProseServiceImpl struct {
	client LLMClient
}

func (s *ProseServiceImpl) GenerateScene(params PromptParams) (*CompletionResponse, error) {
	cfg := PromptRegistry[PromptSceneProse]

	ctx := &compiler.CompiledContext{
		CharacterCards: params.CharacterCards,
		LocationCard:   params.LocationCard,
		Lore:           params.Lore,
		BranchSummary:  params.BranchSummary,
		BeatIntent:     params.BeatIntent,
		POV:            params.POV,
		Tone:           params.Tone,
		TargetWords:    params.TargetWords,
	}
	if params.CharState != nil {
		ctx.CharState = make(map[string]ledger.CharacterState)
		for k, v := range params.CharState {
			b, _ := json.Marshal(v)
			var cs ledger.CharacterState
			if json.Unmarshal(b, &cs) == nil {
				ctx.CharState[k] = cs
			}
		}
	}

	systemPrompt := ctx.BuildSceneProseSystemPrompt()
	userMessage := ctx.BuildSceneProseUserMessage()

	req := CompletionRequest{
		Model:       cfg.Model,
		System:      systemPrompt,
		UserMessage: userMessage,
		Temperature: cfg.Temperature,
		MaxTokens:   4096,
	}

	return s.client.Complete(req)
}

func NewExtractionService(client LLMClient) *ExtractionServiceImpl {
	return &ExtractionServiceImpl{client: client}
}

type ExtractionServiceImpl struct {
	client LLMClient
}

func (s *ExtractionServiceImpl) ExtractState(sceneText string) (map[string]interface{}, error) {
	cfg := PromptRegistry[PromptStateExtract]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      compiler.BuildStateExtractSystemPrompt(),
		UserMessage: sceneText,
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(req)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(res.Content), &result); err != nil {
		result = map[string]interface{}{"raw": res.Content}
	}
	return result, nil
}

func NewSummaryService(client LLMClient) *SummaryServiceImpl {
	return &SummaryServiceImpl{client: client}
}

type SummaryServiceImpl struct {
	client LLMClient
}

func (s *SummaryServiceImpl) UpdateSummary(previousSummary, newScene string) (string, error) {
	cfg := PromptRegistry[PromptSummaryUpdate]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildSummaryUpdateSystemPrompt(previousSummary, newScene),
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(req)
	if err != nil {
		return "", err
	}
	return res.Content, nil
}

func NewMergeService(client LLMClient) *MergeServiceImpl {
	return &MergeServiceImpl{client: client}
}

type MergeServiceImpl struct {
	client LLMClient
}

func (s *MergeServiceImpl) MergeBranches(summaryA, summaryB, timelineNote string) (map[string]interface{}, error) {
	cfg := PromptRegistry[PromptJoinMerge]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildJoinMergeSystemPrompt(summaryA, summaryB, timelineNote),
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(req)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(res.Content), &result); err != nil {
		return nil, fmt.Errorf("merge: JSON parse error: %w\nraw: %s", err, res.Content)
	}
	return result, nil
}

func NewValidationService(client LLMClient) *ValidationServiceImpl {
	return &ValidationServiceImpl{client: client}
}

type ValidationServiceImpl struct {
	client LLMClient
}

func (s *ValidationServiceImpl) ValidateAgainstCanon(canonXML, charState, draft string) (map[string]interface{}, error) {
	cfg := PromptRegistry[PromptCanonValidate]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildCanonValidateSystemPrompt(canonXML, charState, draft),
		Temperature: cfg.Temperature,
		MaxTokens:   2048,
	}
	res, err := s.client.Complete(req)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(res.Content), &result); err != nil {
		return nil, fmt.Errorf("validate: JSON parse error: %w\nraw: %s", err, res.Content)
	}
	return result, nil
}

func NewOutlineService(client LLMClient) *OutlineServiceImpl {
	return &OutlineServiceImpl{client: client}
}

type OutlineServiceImpl struct {
	client LLMClient
}

func (s *OutlineServiceImpl) GenerateOutline(synopsis string) (*StoryOutline, error) {
	cfg := PromptRegistry[PromptOutlineStory]
	systemPrompt := compiler.BuildOutlineStorySystemPrompt(synopsis)
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      systemPrompt,
		UserMessage: synopsis,
		Temperature: cfg.Temperature,
		MaxTokens:   4096,
	}
	res, err := s.client.Complete(req)
	if err != nil {
		return nil, fmt.Errorf("outline: %w", err)
	}
	var outline StoryOutline
	if err := json.Unmarshal([]byte(res.Content), &outline); err != nil {
		return nil, fmt.Errorf("outline: JSON parse error: %w\nraw: %s", err, res.Content)
	}
	return &outline, nil
}


