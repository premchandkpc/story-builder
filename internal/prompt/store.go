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
	clone := *tmpl
	clone.ID = uuid.New()
	s.data[tmpl.Name] = &clone
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
			MaxTokens:   2048,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: "You are a master story architect. Given a synopsis, generate a structured story outline with characters, plot beats, and narrative flow.", Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
				{ID: LayerScene, Strategy: MergeAppend, System: `
Output ONLY valid JSON. No markdown. No code fences. No commentary.

EXAMPLE:
{"title":"The Heist","synopsis":"A crew plans a museum heist.","characters":[{"name":"Max","persona":"Leader","backstory":"Ex-con","moral_alignment":"neutral"},{"name":"Lena","persona":"Hacker","backstory":"Whiz kid","moral_alignment":"good"}],"beats":[{"title":"The Plan","beat_intent":"Setup","character_names":["Max","Lena"],"pov":"Max","tone":"tense","target_words":500,"act":1},{"title":"The Job","beat_intent":"Action","character_names":["Max","Lena"],"pov":"Lena","tone":"suspenseful","target_words":500,"act":1}],"edges":[{"from":"The Plan","to":"The Job","type":"seq"}]}

YOUR TASK: Generate a new outline for the given synopsis.

SCHEMA (use exactly these fields):
- title: string
- synopsis: string
- characters: array of {name, persona, backstory, moral_alignment}
- beats: array of {title, beat_intent, character_names[], pov, tone, target_words (400-1000), act (1-3)}
- edges: array of {from, to, type (seq/fork/join)}

RULES:
1. 5-8 beats. First beat = inciting incident. Last beat = climax.
2. Edge from/to must match beat titles exactly.
3. Every key and string value MUST have opening AND closing double quotes.
4. No trailing commas. No comments.`, Priority: 3},
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
		{
			Name:        "generate_bible",
			Model:       "claude-sonnet",
			Temperature: 0.3,
			MaxTokens:   8192,
			Layers: []PromptLayer{
				{ID: LayerGlobal, Strategy: MergeOverride, System: `You are a world-building expert. Generate a Story Bible as JSON.

OUTPUT SCHEMA (valid JSON only, no markdown, no code fences):
{
  "title": "string (story title)",
  "world": "string (rich description of the world setting)",
  "dimensions": [
    {
      "name": "string",
      "description": "string",
      "physics": "string (optional)",
      "timeFlow": "string (optional)"
    }
  ],
  "worldRules": [
    {
      "category": "string (e.g. physics, magic, society)",
      "description": "string",
      "strictness": "string (absolute|firm|flexible)"
    }
  ],
  "magicSystems": [
    {
      "name": "string",
      "source": "string (where magic comes from)",
      "cost": "string (what it costs to use)",
      "limitations": ["string"],
      "users": ["string (who can use it)"]
    }
  ],
  "factions": [
    {
      "name": "string",
      "goal": "string",
      "resources": "string (optional)",
      "members": ["string (character names, optional)"],
      "relations": "string (optional, how they relate to others)"
    }
  ],
  "cultures": [
    {
      "name": "string",
      "values": ["string"],
      "customs": ["string"],
      "technology": "string (optional)",
      "government": "string (optional)"
    }
  ],
  "tone": "string (narrative tone description)",
  "centralTheme": "string",
  "narrativeVoice": "string (e.g. third-person limited, omniscient)"
}

RULES:
1. World must be internally consistent — every rule, faction, and culture must align.
2. Characters from the outline should appear in relevant factions/cultures/magic users.
3. 10k-50k tokens worth of detail. Be rich but structured.
4. Output ONLY the JSON object. Nothing else.`, Priority: 1},
				{ID: LayerFrame, Strategy: MergeAppend, Priority: 2},
			},
		},
	}
}
