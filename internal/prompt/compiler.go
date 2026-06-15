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
	safetyLayer := ""
	var userLayer *PromptLayer
	frameLayer := &PromptLayer{}
	appliedLayers := make([]LayerID, 0, len(layers))
	model := tmpl.Model
	temp := tmpl.Temperature
	maxTokens := tmpl.MaxTokens

	for _, layer := range layers {
		switch layer.Strategy {
		case MergeDisable:
			continue
		case MergeOverride:
			if layer.HasUserContent() {
				userLayer = &layer
			} else if layer.System != "" {
				systemParts = []string{layer.System}
			}
		case MergeReplace:
			if layer.System != "" {
				systemParts = []string{layer.System}
			}
		case MergeMerge:
			if layer.System != "" {
				systemParts = append(systemParts, layer.System)
			}
		case MergeAppend:
			if layer.System != "" {
				systemParts = append(systemParts, layer.System)
			}
		}

		if layer.ID == LayerSafety {
			safetyLayer = layer.System
		}
		if layer.ID == LayerFrame {
			frameLayer = &layer
		}
		if layer.Model != "" {
			model = layer.Model
		}

		appliedLayers = append(appliedLayers, layer.ID)
	}

	system := strings.Join(systemParts, "\n\n")
	if len(systemParts) > 0 {
		system += "\n\n"
	}
	system += s.buildUserPrompt(req, frameLayer)

	if safetyLayer != "" {
		system = safetyLayer + "\n\n" + system
	}

	user := req.ScenePrompt
	if userLayer != nil && userLayer.Template != "" {
		user = userLayer.Template
	} else if user == "" {
		user = "Write the scene."
	}

	return &CompiledPrompt{
		System:         system,
		User:           user,
		Model:          model,
		Temperature:    temp,
		MaxTokens:      maxTokens,
		LayersApplied:  appliedLayers,
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

func (s *CompilerService) buildUserPrompt(req *CompileRequest, frameLayer *PromptLayer) string {
	var b strings.Builder

	if req.StoryPrompt != "" {
		b.WriteString(fmt.Sprintf("<story>%s</story>\n", esc(req.StoryPrompt)))
	}
	if req.ChapterPrompt != "" {
		b.WriteString(fmt.Sprintf("<chapter>%s</chapter>\n", esc(req.ChapterPrompt)))
	}
	if req.CulturePrompt != "" {
		b.WriteString(fmt.Sprintf("<culture>%s</culture>\n", esc(req.CulturePrompt)))
	}
	if req.MemoryContext != "" {
		b.WriteString(fmt.Sprintf("<memory>%s</memory>\n", esc(req.MemoryContext)))
	}
	if req.CharacterPrompt != "" {
		b.WriteString(fmt.Sprintf("<character_prompt>%s</character_prompt>\n", esc(req.CharacterPrompt)))
	}

	return b.String()
}

func (p *PromptLayer) HasUserContent() bool {
	return p.Template != ""
}

func esc(s string) string {
	return strings.NewReplacer("<", "＜", ">", "＞").Replace(s)
}
