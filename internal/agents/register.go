package agents

import "github.com/premchand/story-builder/internal/llm"

func RegisterAll(registry *AgentRegistry, llmClient llm.LLMClient, proseSvc llm.ProseService, extractSvc llm.ExtractionService, validateSvc llm.ValidationService) {
	specs := []AgentSpec{
		NewDirectorSpec(llmClient, proseSvc),
		NewCharacterSpec(llmClient, proseSvc),
		NewNarratorSpec(llmClient, proseSvc),
		NewCanonGuardSpec(llmClient, validateSvc),
		NewEditorSpec(llmClient),
		NewCriticSpec(llmClient),
		NewStateExtractorSpec(llmClient, extractSvc),
		NewWorldSpec(llmClient),
		NewArcSpec(llmClient),
		NewMemorySpec(llmClient),
	}
	for _, spec := range specs {
		registry.Register(spec)
	}
}
