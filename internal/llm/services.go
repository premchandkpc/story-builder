package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/ledger"
)

func NewProseService(client LLMClient) *ProseServiceImpl {
	return &ProseServiceImpl{client: client}
}

type ProseServiceImpl struct {
	client LLMClient
}

func (s *ProseServiceImpl) GenerateScene(ctx context.Context, params PromptParams) (*CompletionResponse, error) {
	cfg := PromptRegistry[PromptSceneProse]

	cc := &compiler.CompiledContext{
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
		cc.CharState = make(map[string]ledger.CharacterState)
		for k, v := range params.CharState {
			b, _ := json.Marshal(v)
			var cs ledger.CharacterState
			if json.Unmarshal(b, &cs) == nil {
				cc.CharState[k] = cs
			}
		}
	}

	systemPrompt := cc.BuildSceneProseSystemPrompt()
	userMessage := cc.BuildSceneProseUserMessage()

	req := CompletionRequest{
		Model:       cfg.Model,
		System:      systemPrompt,
		UserMessage: userMessage,
		Temperature: cfg.Temperature,
		MaxTokens:   4096,
	}

	return s.client.Complete(ctx, req)
}

func NewExtractionService(client LLMClient) *ExtractionServiceImpl {
	return &ExtractionServiceImpl{client: client}
}

type ExtractionServiceImpl struct {
	client LLMClient
}

func (s *ExtractionServiceImpl) ExtractState(ctx context.Context, sceneText string, roster map[string]string) (*ledger.StateDeltas, error) {
	cfg := PromptRegistry[PromptStateExtract]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      compiler.BuildStateExtractSystemPrompt(roster),
		UserMessage: sceneText,
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	var result ledger.StateDeltas
	if err := parseJSONPayload(res.Content, &result); err != nil {
		return nil, fmt.Errorf("extract state: %w", err)
	}
	if result.Deltas == nil {
		result.Deltas = []ledger.StateDelta{}
	}
	return &result, nil
}

func NewSummaryService(client LLMClient) *SummaryServiceImpl {
	return &SummaryServiceImpl{client: client}
}

type SummaryServiceImpl struct {
	client LLMClient
}

func (s *SummaryServiceImpl) UpdateSummary(ctx context.Context, previousSummary, newScene string) (string, error) {
	cfg := PromptRegistry[PromptSummaryUpdate]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildSummaryUpdateSystemPrompt(previousSummary, newScene),
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(ctx, req)
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

func (s *MergeServiceImpl) MergeBranches(ctx context.Context, summaryA, summaryB, timelineNote string) (map[string]interface{}, error) {
	cfg := PromptRegistry[PromptJoinMerge]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildJoinMergeSystemPrompt(summaryA, summaryB, timelineNote),
		Temperature: cfg.Temperature,
		MaxTokens:   1024,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := parseJSONPayload(res.Content, &result); err != nil {
		return nil, fmt.Errorf("merge: %w", err)
	}
	return result, nil
}

func NewValidationService(client LLMClient) *ValidationServiceImpl {
	return &ValidationServiceImpl{client: client}
}

type ValidationServiceImpl struct {
	client LLMClient
}

func (s *ValidationServiceImpl) ValidateAgainstCanon(ctx context.Context, canonXML, charState, draft string) (map[string]interface{}, error) {
	cfg := PromptRegistry[PromptCanonValidate]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      "",
		UserMessage: compiler.BuildCanonValidateSystemPrompt(canonXML, charState, draft),
		Temperature: cfg.Temperature,
		MaxTokens:   2048,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := parseJSONPayload(res.Content, &result); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return result, nil
}

func NewOutlineService(client LLMClient) *OutlineServiceImpl {
	return &OutlineServiceImpl{client: client}
}

type OutlineServiceImpl struct {
	client LLMClient
}

func (s *OutlineServiceImpl) GenerateOutline(ctx context.Context, synopsis string) (*StoryOutline, error) {
	cfg := PromptRegistry[PromptOutlineStory]
	systemPrompt := compiler.BuildOutlineStorySystemPrompt(synopsis)
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      systemPrompt,
		UserMessage: synopsis,
		Temperature: cfg.Temperature,
		MaxTokens:   4096,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("outline: %w", err)
	}
	var outline StoryOutline
	if err := parseJSONPayload(res.Content, &outline); err != nil {
		return nil, fmt.Errorf("outline: %w", err)
	}
	return &outline, nil
}

func NewTitleService(client LLMClient) *TitleServiceImpl {
	return &TitleServiceImpl{client: client}
}

type TitleServiceImpl struct {
	client LLMClient
}

func (s *TitleServiceImpl) GenerateTitle(ctx context.Context, synopsis string) (string, error) {
	cfg := PromptRegistry[PromptGenerateTitle]
	req := CompletionRequest{
		Model:       cfg.Model,
		System:      cfg.SystemText,
		UserMessage: synopsis,
		Temperature: cfg.Temperature,
		MaxTokens:   64,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("title: %w", err)
	}
	return strings.TrimSpace(res.Content), nil
}

func parseJSONPayload[T any](content string, out *T) error {
	payload := strings.TrimSpace(content)
	if payload == "" {
		return fmt.Errorf("empty response")
	}
	if strings.HasPrefix(payload, "```") {
		payload = strings.TrimPrefix(payload, "```json")
		payload = strings.TrimPrefix(payload, "```")
		payload = strings.TrimSpace(payload)
		if idx := strings.LastIndex(payload, "```"); idx >= 0 {
			payload = strings.TrimSpace(payload[:idx])
		}
	}
	if !json.Valid([]byte(payload)) {
		start := strings.Index(payload, "{")
		end := strings.LastIndex(payload, "}")
		if start >= 0 && end > start {
			payload = payload[start : end+1]
		}
	}
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON: %s", payload)
	}
	return json.Unmarshal([]byte(payload), out)
}
