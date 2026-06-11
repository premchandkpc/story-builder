package compiler

import "fmt"

func (c *CompiledContext) BuildSceneProseSystemPrompt() string {
	canon := ""

	for _, card := range c.CharacterCards {
		canon += fmt.Sprintf("<character name=\"%s\">\n", card.Name)
		canon += fmt.Sprintf("Traits: %v\n", card.Traits)
		canon += fmt.Sprintf("Relationships: %v\n", card.Relationships)
		canon += "Voice samples:\n"
		for _, v := range card.VoiceSamples {
			canon += fmt.Sprintf("- \"%s\"\n", v)
		}
		canon += "</character>\n"
	}

	if c.LocationCard != nil {
		canon += fmt.Sprintf("<location name=\"%s\">%s\n", c.LocationCard.Name, c.LocationCard.Description)
		canon += fmt.Sprintf("Props available: %v</location>\n", c.LocationCard.Props)
	}

	canon += "<world_rules>\n"
	for _, l := range c.Lore {
		canon += fmt.Sprintf("- %s\n", l)
	}
	canon += "</world_rules>"

	stateBlock := ""
	for char, st := range c.CharState {
		stateBlock += fmt.Sprintf("%s: at %s, mood %s,\n", char, st.Location, st.Mood)
		stateBlock += fmt.Sprintf("knows: %v,\n", st.Knows)
		if st.DoesNotKnow != nil {
			stateBlock += fmt.Sprintf("does NOT know: %v\n", st.DoesNotKnow)
		} else {
			stateBlock += "does NOT know: []\n"
		}
	}

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
		canon, stateBlock, c.BranchSummary, c.TargetWords)
}

func (c *CompiledContext) BuildSceneProseUserMessage() string {
	return fmt.Sprintf("Write the scene where: %s. POV: %s. Tone: %s.",
		c.BeatIntent, c.POV, c.Tone)
}

func BuildStateExtractSystemPrompt() string {
	return `You are a continuity clerk. Read the scene and call record_state_deltas.
Rules: extract ONLY what is explicit in the text. No inference, no
speculation about feelings not shown. If a character appears but nothing
changed for them, omit them entirely. "learned" means information the
character witnessed or was told IN THIS SCENE.`
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
- Output the summary only.`, prevSummary, newScene)
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
in the merged summary.`, summaryA, summaryB, timelineNote)
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
		compiledCanon, charState, draft)
}
