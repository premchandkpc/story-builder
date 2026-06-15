package prompt

import (
	"fmt"
	"sort"
	"strings"
)

type CompilerService struct {
	store Store
}

func NewCompilerService(store Store) *CompilerService {
	return &CompilerService{store: store}
}

var layerOrder = []LayerID{
	LayerGlobal,
	LayerCulture,
	LayerStory,
	LayerMemory,
	LayerChapter,
	LayerScenario,
	LayerCharacter,
	LayerScene,
	LayerFrame,
	LayerSafety,
}

func (s *CompilerService) Compile(req *CompileRequest, templateName string) (*CompiledPrompt, error) {
	tmpl, err := s.store.Get(templateName)
	if err != nil {
		return nil, fmt.Errorf("prompt compiler: get template %q: %w", templateName, err)
	}

	layers := s.resolveLayers(tmpl.Layers)

	var systemParts []string
	var userLayer *PromptLayer
	appliedLayers := make([]LayerID, 0, len(layers))
	model := tmpl.Model
	temp := tmpl.Temperature
	maxTokens := tmpl.MaxTokens

	for _, layer := range layers {
		if layer.Strategy == MergeDisable {
			continue
		}

		content := layer.System
		if layer.ID == LayerFrame {
			content = s.buildDynamicContext(req)
			if content == "" {
				appliedLayers = append(appliedLayers, layer.ID)
				continue
			}
		}

		switch layer.Strategy {
		case MergeOverride:
			if layer.HasUserContent() {
				userLayer = &layer
				continue
			}
			if content != "" {
				systemParts = []string{content}
			}
		case MergeReplace:
			if content != "" {
				systemParts = []string{content}
			}
		case MergeMerge, MergeAppend:
			if content != "" {
				systemParts = append(systemParts, content)
			}
		}

		if layer.Model != "" {
			model = layer.Model
		}
		appliedLayers = append(appliedLayers, layer.ID)
	}

	system := strings.Join(systemParts, "\n\n")

	user := req.ScenePrompt
	if userLayer != nil && userLayer.Template != "" {
		user = userLayer.Template
	} else if user == "" {
		user = "Write the scene."
	}

	return &CompiledPrompt{
		System:        system,
		User:          user,
		Model:         model,
		Temperature:   temp,
		MaxTokens:     maxTokens,
		LayersApplied: appliedLayers,
	}, nil
}

func (s *CompilerService) resolveLayers(layers []PromptLayer) []PromptLayer {
	sorted := make([]PromptLayer, len(layers))
	copy(sorted, layers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return layerIndex(sorted[i].ID) < layerIndex(sorted[j].ID)
	})
	return sorted
}

func layerIndex(id LayerID) int {
	for i, lid := range layerOrder {
		if lid == id {
			return i
		}
	}
	return len(layerOrder)
}

func (s *CompilerService) buildDynamicContext(req *CompileRequest) string {
	var b strings.Builder

	if req.CanonXML != "" {
		b.WriteString(fmt.Sprintf("<canon>\n%s\n</canon>", req.CanonXML))
	}
	if req.CharStateXML != "" {
		b.WriteString(fmt.Sprintf("\n\n<current_state>\n%s\n</current_state>", req.CharStateXML))
	}
	if req.BranchSummary != "" {
		b.WriteString(fmt.Sprintf("\n\n<story_so_far>%s</story_so_far>", esc(req.BranchSummary)))
	}
	if req.TargetWords > 0 {
		b.WriteString(fmt.Sprintf("\n\nTarget word count: %d.", req.TargetWords))
	}
	if req.RosterJSON != "" {
		b.WriteString(fmt.Sprintf("\nKnown characters and names to preserve: %s", req.RosterJSON))
	}
	if req.Synopsis != "" {
		b.WriteString(fmt.Sprintf("\n<synopsis>%s</synopsis>", esc(req.Synopsis)))
	}

	if req.StoryPrompt != "" {
		b.WriteString(fmt.Sprintf("\n<story>%s</story>", esc(req.StoryPrompt)))
	}
	if req.ChapterPrompt != "" {
		b.WriteString(fmt.Sprintf("\n<chapter>%s</chapter>", esc(req.ChapterPrompt)))
	}
	if req.CulturePrompt != "" {
		b.WriteString(fmt.Sprintf("\n<culture>%s</culture>", esc(req.CulturePrompt)))
	}
	if req.MemoryContext != "" {
		b.WriteString(fmt.Sprintf("\n<memory>%s</memory>", esc(req.MemoryContext)))
	}
	if req.CharacterPrompt != "" {
		b.WriteString(fmt.Sprintf("\n<character_prompt>%s</character_prompt>", esc(req.CharacterPrompt)))
	}
	return b.String()
}

func (p *PromptLayer) HasUserContent() bool {
	return p.Template != ""
}

func esc(s string) string {
	return strings.NewReplacer("<", "＜", ">", "＞").Replace(s)
}
