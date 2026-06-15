package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/ledger"
	"github.com/premchand/story-builder/internal/prompt"
)

func NewProseService(client LLMClient, c *prompt.CompilerService) *ProseServiceImpl {
	return &ProseServiceImpl{client: client, compiler: c}
}

type ProseServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *ProseServiceImpl) GenerateScene(ctx context.Context, params PromptParams) (*CompletionResponse, error) {
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

	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt:   cc.BuildSceneProseUserMessage(),
		CanonXML:      cc.BuildCanonXML(),
		CharStateXML:  cc.BuildCharStateXML(),
		BranchSummary: params.BranchSummary,
		TargetWords:   params.TargetWords,
	}, "scene_prose")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}

	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
	}
	return s.client.Complete(ctx, req)
}

func NewExtractionService(client LLMClient, c *prompt.CompilerService) *ExtractionServiceImpl {
	return &ExtractionServiceImpl{client: client, compiler: c}
}

type ExtractionServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *ExtractionServiceImpl) ExtractState(ctx context.Context, sceneText string, roster map[string]string) (*ledger.StateDeltas, error) {
	rosterJSON, _ := json.Marshal(roster)
	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt: sceneText,
		RosterJSON:  string(rosterJSON),
	}, "state_extract")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}
	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
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

func NewSummaryService(client LLMClient, c *prompt.CompilerService) *SummaryServiceImpl {
	return &SummaryServiceImpl{client: client, compiler: c}
}

type SummaryServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *SummaryServiceImpl) UpdateSummary(ctx context.Context, previousSummary, newScene string) (string, error) {
	userMsg := compiler.BuildSummaryUpdateSystemPrompt(previousSummary, newScene)
	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt: userMsg,
	}, "summary_update")
	if err != nil {
		return "", fmt.Errorf("compile prompt: %w", err)
	}
	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Content, nil
}

func NewMergeService(client LLMClient, c *prompt.CompilerService) *MergeServiceImpl {
	return &MergeServiceImpl{client: client, compiler: c}
}

type MergeServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *MergeServiceImpl) MergeBranches(ctx context.Context, summaryA, summaryB, timelineNote string) (map[string]interface{}, error) {
	userMsg := compiler.BuildJoinMergeSystemPrompt(summaryA, summaryB, timelineNote)
	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt: userMsg,
	}, "join_merge")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}
	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
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

func NewValidationService(client LLMClient, c *prompt.CompilerService) *ValidationServiceImpl {
	return &ValidationServiceImpl{client: client, compiler: c}
}

type ValidationServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *ValidationServiceImpl) ValidateAgainstCanon(ctx context.Context, canonXML, charState, draft string) (map[string]interface{}, error) {
	userMsg := compiler.BuildCanonValidateSystemPrompt(canonXML, charState, draft)
	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt:   userMsg,
		CompiledCanon: canonXML,
	}, "canon_validate")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}
	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
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

func NewOutlineService(client LLMClient, c *prompt.CompilerService) *OutlineServiceImpl {
	return &OutlineServiceImpl{client: client, compiler: c}
}

type OutlineServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func (s *OutlineServiceImpl) GenerateOutline(ctx context.Context, synopsis string) (*StoryOutline, error) {
	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt: synopsis,
		Synopsis:    synopsis,
	}, "outline_story")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}
	req := CompletionRequest{
		Model:       ModelTier(compiled.Model),
		System:      compiled.System,
		UserMessage: compiled.User,
		Temperature: compiled.Temperature,
		MaxTokens:   compiled.MaxTokens,
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
