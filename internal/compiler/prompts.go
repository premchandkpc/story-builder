package compiler

import (
	"fmt"
	"strings"
)

func esc(s string) string {
	return strings.NewReplacer("<", "＜", ">", "＞").Replace(s)
}

func (c *CompiledContext) BuildCanonXML() string {
	canon := ""
	for _, card := range c.CharacterCards {
		canon += fmt.Sprintf("<character name=\"%s\">\n", esc(card.Name))
		canon += fmt.Sprintf("Traits: %v\n", card.Traits)
		canon += fmt.Sprintf("Relationships: %v\n", card.Relationships)
		canon += "Voice samples:\n"
		for _, v := range card.VoiceSamples {
			canon += fmt.Sprintf("- \"%s\"\n", esc(v))
		}
		canon += "</character>\n"
	}
	if c.LocationCard != nil {
		canon += fmt.Sprintf("<location name=\"%s\">%s\n", esc(c.LocationCard.Name), esc(c.LocationCard.Description))
		canon += fmt.Sprintf("Props available: %v</location>\n", c.LocationCard.Props)
	}
	canon += "<world_rules>\n"
	for _, l := range c.Lore {
		canon += fmt.Sprintf("- %s\n", esc(l))
	}
	canon += "</world_rules>"
	return canon
}

func (c *CompiledContext) BuildCharStateXML() string {
	stateBlock := ""
	for char, st := range c.CharState {
		stateBlock += fmt.Sprintf("%s: at %s, mood %s,\n", esc(char), esc(st.Location), esc(st.Mood))
		stateBlock += fmt.Sprintf("knows: %v,\n", st.Knows)
		if st.DoesNotKnow != nil {
			stateBlock += fmt.Sprintf("does NOT know: %v\n", st.DoesNotKnow)
		} else {
			stateBlock += "does NOT know: []\n"
		}
	}
	return stateBlock
}

func (c *CompiledContext) BuildSceneProseSystemPrompt() string {
	return fmt.Sprintf(`You are a fiction co-writer. Write ONE scene and nothing else.

<canon>
%s
</canon>

<current_state>
%s
</current_state>

<story_so_far>%s</story_so_far>

HARD RULES:
1. Canon is law. Never contradict <canon> or <current_state>.
2. A character cannot reference knowledge listed in their "does NOT know".
3. Introduce NO new named characters or locations. Unnamed extras are fine.
4. Every line of dialogue must pass the voice-sample test for that character.
5. End the scene when the beat resolves. Do not set up the next scene.
6. Length: %d words, ±20%%.
7. Output prose only — no titles, no notes, no "Scene:" headers.`,
		c.BuildCanonXML(), c.BuildCharStateXML(), esc(c.BranchSummary), c.TargetWords)
}

func (c *CompiledContext) BuildSceneProseUserMessage() string {
	return fmt.Sprintf("Write the scene where: %s. POV: %s. Tone: %s.",
		esc(c.BeatIntent), esc(c.POV), esc(c.Tone))
}

func BuildStateExtractSystemPrompt(roster map[string]string) string {
	rosterBlock := ""
	if len(roster) > 0 {
		rosterBlock = "\nKnown characters and names to preserve: " + strings.Join(mapValues(roster), ", ") + ""
	}
	return fmt.Sprintf(`You are a continuity clerk. Read the scene and call record_state_deltas.
Rules: extract ONLY what is explicit in the text. No inference, no
speculation about feelings not shown. If a character appears but nothing
changed for them, omit them entirely. "learned" means information the
character witnessed or was told IN THIS SCENE.%s
Output valid JSON with a top-level object containing a "deltas" array and an optional "open_threads" array.`, rosterBlock)
}

func mapValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

func BuildSummaryUpdateSystemPrompt(prevSummary, newScene string) string {
	return fmt.Sprintf(`You maintain a running plot summary for one storyline branch.

<previous_summary>%s</previous_summary>
<new_scene>%s</new_scene>

Produce an updated summary. Rules:
- Max 200 words. Plot facts and character knowledge only — no prose style,
  no atmosphere, no quotes.
- Preserve every fact from the previous summary unless the new scene
  explicitly supersedes it.
- Chronological order. Present tense.
- Output the summary only.`, esc(prevSummary), esc(newScene))
}

func BuildJoinMergeSystemPrompt(summaryA, summaryB, timelineNote string) string {
	return fmt.Sprintf(`Two parallel storylines are converging. Merge their summaries.

<branch_a>%s</branch_a>
<branch_b>%s</branch_b>
<timeline_note>%s</timeline_note>

Output JSON: {"merged_summary": "...", "conflicts": [{"description": "...",
"severity": "blocking|warning"}]}

A conflict is: the same character acting in both branches, contradictory
facts, or events that cannot coexist on the stated timeline. If branches
are cleanly disjoint, conflicts is []. Interleave events chronologically
in the merged summary.`, esc(summaryA), esc(summaryB), esc(timelineNote))
}

func BuildCanonValidateSystemPrompt(compiledCanon, charState, draft string) string {
	return fmt.Sprintf(`You are a strict continuity editor. Check this draft against canon.

<canon>%s</canon>
<current_state>%s</current_state>
<draft>%s</draft>

Output JSON: {"violations": [{"type": "voice|knowledge|trait|location|world_rule",
"character": "...", "evidence": "<short quote from draft>",
"explanation": "...", "severity": "high|low"}]}

Check specifically: (1) any character using knowledge from their does-not-know
list, (2) dialogue that doesn't match voice samples, (3) trait contradictions,
(4) physical impossibilities given locations, (5) world-rule breaks.
Empty array if clean. Do not comment on writing quality — continuity only.`,
		esc(compiledCanon), esc(charState), esc(draft))
}

func BuildOutlineStorySystemPrompt(synopsis string) string {
	return fmt.Sprintf(`You are a master story architect. Given a synopsis, generate a structured story outline.

<synopsis>%s</synopsis>

Output VALID JSON only — no markdown, no code fences, no commentary. Schema:
{
  "title": "...",
  "synopsis": "...",
  "characters": [
    {
      "name": "...",
      "persona": "archetype and role",
      "backstory": "2-3 sentence backstory",
      "moral_alignment": "good|neutral|evil|ambiguous",
      "personality": ["trait1", "trait2"],
      "flaws": ["flaw1"],
      "goals": ["goal1", "goal2"],
      "voice_samples": ["sample line 1", "sample line 2"]
    }
  ],
  "beats": [
    {
      "title": "scene title",
      "beat_intent": "what happens in this scene",
      "character_names": ["char1", "char2"],
      "location_name": "where (optional)",
      "pov": "POV character name",
      "tone": "mood",
      "target_words": 750,
      "act": 1
    }
  ],
  "edges": [
    { "from": "scene title", "to": "next scene title", "type": "seq" }
  ]
}

RULES:
1. 5-12 beats. First beat = inciting incident. Last beat = climax + resolution.
2. Each character has at least one goal and one flaw.
3. Character names must exactly match across beats, edges, and characters array.
4. Edge types: seq (scene follows previous), fork (branch point), join (convergence).
5. Assign acts (1-3) so each act has 2-5 beats.
6. target_words per beat: 400-1000.
7. Provide voice_samples (2 per character) that reveal personality.`,
		esc(synopsis))
}
