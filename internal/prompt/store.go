package prompt

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]*PromptTemplate
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]*PromptTemplate),
	}
}

func (s *MemoryStore) Save(tmpl *PromptTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmpl.ID = uuid.New()
	s.data[tmpl.Name] = tmpl
	return nil
}

func (s *MemoryStore) Get(name string) (*PromptTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tmpl, ok := s.data[name]
	if !ok {
		return nil, fmt.Errorf("prompt template %q not found", name)
	}
	return tmpl, nil
}

func (s *MemoryStore) List() ([]PromptTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PromptTemplate, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, *v)
	}
	return out, nil
}

func (s *MemoryStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, name)
	return nil
}

func DefaultTemplates() []*PromptTemplate {
	return []*PromptTemplate{
		{
			Name:        "scene_prose",
			Model:       "claude-sonnet",
			Temperature: 0.8,
			MaxTokens:   4096,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a fiction co-writer. Write ONE scene and nothing else.", Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
				{ID: LayerSafety, Strategy: MergeAppend, System: "HARD RULES:\n1. Canon is law. Never contradict <canon> or <current_state>.\n2. A character cannot reference knowledge listed in their 'does NOT know'.\n3. Introduce NO new named characters or locations. Unnamed extras are fine.\n4. Every line of dialogue must pass the voice-sample test for that character.\n5. End the scene when the beat resolves. Do not set up the next scene.\n6. Output prose only — no titles, no notes, no 'Scene:' headers.", Priority: 10},
			},
		},
		{
			Name:        "state_extract",
			Model:       "local-7b",
			Temperature: 0.0,
			MaxTokens:   2048,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a continuity clerk. Read the scene and call record_state_deltas.", Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
				{ID: LayerScene, Strategy: MergeAppend, System: "Rules: extract ONLY what is explicit in the text. No inference, no speculation about feelings not shown. If a character appears but nothing changed for them, omit them entirely. 'learned' means information the character witnessed or was told IN THIS SCENE.\nOutput valid JSON with a top-level object containing a 'deltas' array and an optional 'open_threads' array.", Priority: 3},
			},
		},
		{
			Name:        "summary_update",
			Model:       "local-7b",
			Temperature: 0.2,
			MaxTokens:   1024,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You maintain a running plot summary for one storyline branch.", Priority: 1},
				{ID: LayerScene, Strategy: MergeAppend, System: "Produce an updated summary. Rules:\n- Max 200 words. Plot facts and character knowledge only — no prose style, no atmosphere, no quotes.\n- Preserve every fact from the previous summary unless the new scene explicitly supersedes it.\n- Chronological order. Present tense.\n- Output the summary only.", Priority: 2},
			},
		},
		{
			Name:        "join_merge",
			Model:       "claude-haiku",
			Temperature: 0.2,
			MaxTokens:   1024,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "Two parallel storylines are converging. Merge their summaries.", Priority: 1},
				{ID: LayerScene, Strategy: MergeAppend, System: "Output JSON: {\"merged_summary\": \"...\", \"conflicts\": [{\"description\": \"...\", \"severity\": \"blocking|warning\"}]}\n\nA conflict is: the same character acting in both branches, contradictory facts, or events that cannot coexist on the stated timeline. If branches are cleanly disjoint, conflicts is []. Interleave events chronologically in the merged summary.", Priority: 2},
			},
		},
		{
			Name:        "canon_validate",
			Model:       "claude-haiku",
			Temperature: 0.0,
			MaxTokens:   2048,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a strict continuity editor. Check this draft against canon.", Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
				{ID: LayerScene, Strategy: MergeAppend, System: "Output JSON: {\"violations\": [{\"type\": \"voice|knowledge|trait|location|world_rule\", \"character\": \"...\", \"evidence\": \"<short quote from draft>\", \"explanation\": \"...\", \"severity\": \"high|low\"}]}\n\nCheck specifically: (1) any character using knowledge from their does-not-know list, (2) dialogue that doesn't match voice samples, (3) trait contradictions, (4) physical impossibilities given locations, (5) world-rule breaks.\nEmpty array if clean. Do not comment on writing quality — continuity only.", Priority: 3},
			},
		},
		{
			Name:        "outline_story",
			Model:       "local-7b",
			Temperature: 0.7,
			MaxTokens:   4096,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a master story architect. Given a synopsis, generate a structured story outline with characters, plot beats, and narrative flow.", Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
				{ID: LayerScene, Strategy: MergeAppend, System: `
Output ONLY valid JSON. No markdown. No code fences. No commentary.

SCHEMA:
{"title":"...","synopsis":"...","characters":[{"name":"...","persona":"...","backstory":"...","moral_alignment":"...","personality":["..."],"flaws":["..."],"goals":["..."],"voice_samples":["...","..."]}],"beats":[{"title":"...","beat_intent":"...","character_names":["..."],"pov":"...","tone":"...","target_words":600,"act":1}],"edges":[{"from":"...","to":"...","type":"seq"}]}

RULES:
1. 5-12 beats. First beat = inciting incident. Last beat = climax + resolution.
2. Character fields: name, persona, backstory, moral_alignment, personality (array), flaws (array), goals (array), voice_samples (2 strings).
3. Beat fields: title, beat_intent, character_names (array), pov, tone, target_words (400-1000), act (1-3).
4. Edge types: seq, fork, join. from/to match beat titles.
5. EVERY string value MUST have opening AND closing double quotes.`, Priority: 3},
			},
		},
		{
			Name:        "generate_title",
			Model:       "local-7b",
			Temperature: 0.5,
			MaxTokens:   64,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a creative title generator. Given a synopsis, generate a short, engaging story title (3-8 words). Return ONLY the title, no quotes or punctuation.", Priority: 1},
			},
		},
	}
}
