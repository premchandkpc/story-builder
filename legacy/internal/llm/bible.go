package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/prompt"
)

type BibleService interface {
	GenerateBible(ctx context.Context, storyID, synopsis string, characters []*domain.Character) (*domain.StoryBible, error)
}

type BibleServiceImpl struct {
	client   LLMClient
	compiler *prompt.CompilerService
}

func NewBibleService(client LLMClient, compiler *prompt.CompilerService) *BibleServiceImpl {
	return &BibleServiceImpl{client: client, compiler: compiler}
}

func (s *BibleServiceImpl) GenerateBible(ctx context.Context, storyID, synopsis string, characters []*domain.Character) (*domain.StoryBible, error) {
	charJSON, _ := json.Marshal(characters)

	compiled, err := s.compiler.Compile(&prompt.CompileRequest{
		ScenePrompt: fmt.Sprintf(`Generate a Story Bible for the following narrative:

SYNOPSIS:
%s

CHARACTERS:
%s`, synopsis, string(charJSON)),
		Synopsis: synopsis,
	}, "generate_bible")
	if err != nil {
		return nil, fmt.Errorf("compile prompt: %w", err)
	}

	req := CompletionRequest{
		Model:        ModelTier(compiled.Model),
		System:       compiled.System,
		UserMessage:  compiled.User,
		Temperature:  compiled.Temperature,
		MaxTokens:    8192,
		ValidateJSON: true,
	}
	res, err := s.client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("bible generation: %w", err)
	}

	var bible domain.StoryBible
	if err := parseJSONPayload(res.Content, &bible); err != nil {
		return nil, fmt.Errorf("bible parse: %w", err)
	}
	bible.StoryID = storyID
	return &bible, nil
}
