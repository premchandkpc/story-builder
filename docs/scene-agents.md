# Multi-Agent Scene System

## Concept

A scene (`Node`) is not one LLM call — it's an interactive process where multiple character agents act within a location/setting. The scene's own "situation flow" definition determines who acts when, in what order, and whether actions are sequential or parallel.

## Scene structure (per node)

Each `Node` gets a `scene_structure` field describing how agents participate:

```
scene_structure: {
  "flow_type": "dialogue" | "monologue" | "parallel" | "round_robin" | "custom",
  "character_order": ["char_id_1", "char_id_2"],   // explicit turn order (optional)
  "situation_flow": "Arthur enters the throne room. Morgaine is already waiting. They argue about the crown before a guard interrupts.",  // natural language flow
  "max_turns": 10                                    // optional cap
}
```

- `flow_type` determines the default orchestration strategy
- `character_order` overrides when specific sequence matters
- `situation_flow` is a narrative description of how the scene should progress (used to prompt agents)

## Turn loop

```
1. Determine next actor(s) based on scene_structure + current turn context
2. Build agent prompt for that character(s):
   - Character card (traits, voice, relationships)
   - Current state (mood, location, knows/does_not_know)
   - Recent context (previous turns in this scene)
   - Situation flow reminder
   - Beat intent (what this scene should accomplish)
3. LLM generates character action/dialogue
   - Default: local 7B (fast, cheap) for each turn
   - Optionally: Sonnet for critical turns
4. Save as SceneTurn
5. User reviews → clicks "next" or "finish scene"
```

## SceneTurn table

| Column        | Type      | Notes                    |
|---------------|-----------|--------------------------|
| id            | uuid      | PK                       |
| node_id       | uuid      | FK -> nodes              |
| turn_number   | int       | Order within scene       |
| actor_ids     | uuid[]    | Characters acting this turn |
| prompt        | text      | Full prompt sent         |
| output        | text      | LLM response             |
| model         | text      | Which model was used     |
| status        | text      | pending/done/accepted/rejected |
| created_at    | timestamptz |                         |

## When is scene "done"?

User decides via frontend — explicit "Finish scene" button.

On finish:
1. Assemble all SceneTurns in order → final scene text
2. Save as a `generation` row (for acceptance/rejection)
3. Run state extraction + summary update (existing pipeline)
4. Mark node status as `generated`

## Keeping both flows

The existing single-shot Sonnet generation stays for:
- Quick drafts / outlines
- Nodes with `scene_structure.flow_type = "monologue"` (single POV)
- Users who want fast results

The multi-agent flow activates when `scene_structure` has `flow_type` other than `monologue`, or when explicitly selected.

Both use the same `generations` table for final output. SceneTurns are intermediate artifacts.

## API endpoints (new)

```
POST /api/v1/stories/{storyID}/nodes/{id}/scene/start   → start multi-agent scene, return first turn prompt
POST /api/v1/stories/{storyID}/nodes/{id}/scene/next    → generate next turn
POST /api/v1/stories/{storyID}/nodes/{id}/scene/finish  → assemble turns → generation, run extraction
GET  /api/v1/stories/{storyID}/nodes/{id}/scene/turns   → list all turns for this scene
PUT  /api/v1/stories/{storyID}/nodes/{id}/scene/structure → set scene_structure
```

## Agent prompt template

```
You are {character_name}. {traits_and_voice}

Current situation: {scene_setting}
Your mood: {mood}
What you know: {knows}
What you don't know: {does_not_know}
Recent events: {previous_narrative_context}

Scene context: {situation_flow}
Scene goal: {beat_intent}

{character_name}'s action:
```

## Next steps (if approved)

1. Migration: add `scene_structure` column to `nodes`, create `scene_turns` table
2. Regenerate sqlc queries
3. Add `scene_structure` to `graph.Node` type
4. Create `internal/scene/` package with turn orchestration logic
5. Add API handlers for start/next/finish/turns
6. Wire prompts per character agent
7. Frontend: turn-by-turn view with advance/finish controls
