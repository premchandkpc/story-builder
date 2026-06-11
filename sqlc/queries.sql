-- name: CreateCharacter :one
INSERT INTO characters (name, persona, backstory, moral_alignment, personality, flaws, goals, traits, voice_samples, relationships, parent_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetCharacterLatest :one
SELECT * FROM latest_characters WHERE id = $1;

-- name: GetCharacterAtVersion :one
SELECT * FROM characters WHERE id = $1 AND version = $2;

-- name: UpdateCharacter :one
INSERT INTO characters (id, version, name, persona, backstory, moral_alignment, personality, flaws, goals, traits, voice_samples, relationships, parent_id)
SELECT c.id, MAX(c.version) + 1, $2, $3, $4, $5, CAST($6 AS jsonb), CAST($7 AS jsonb), CAST($8 AS jsonb), CAST($9 AS jsonb), $10, CAST($11 AS jsonb), $12
FROM characters c WHERE c.id = $1
GROUP BY c.id
RETURNING *;

-- name: ListCharacters :many
SELECT * FROM latest_characters ORDER BY name;

-- name: CreateActor :one
INSERT INTO actors (name, gender, ethnicity, race, skin_tone, eye_color, hair_color, hair_style, build, height_cm, weight_kg, age, nationality, traits)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetActor :one
SELECT * FROM actors WHERE id = $1;

-- name: UpdateActor :one
UPDATE actors SET
  name = $2, gender = $3, ethnicity = $4, race = $5,
  skin_tone = $6, eye_color = $7, hair_color = $8, hair_style = $9,
  build = $10, height_cm = $11, weight_kg = $12, age = $13,
  nationality = $14, traits = CAST($15 AS jsonb)
WHERE id = $1
RETURNING *;

-- name: ListActors :many
SELECT * FROM actors ORDER BY name;

-- name: CreateCharacterTrait :one
INSERT INTO character_traits (name, category, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCharacterTrait :one
SELECT * FROM character_traits WHERE id = $1;

-- name: ListCharacterTraits :many
SELECT * FROM character_traits ORDER BY name;

-- name: AssignTrait :exec
INSERT INTO character_trait_assignments (character_id, trait_id, intensity, note)
VALUES ($1, $2, $3, $4)
ON CONFLICT (character_id, trait_id) DO UPDATE SET intensity = $3, note = $4;

-- name: UnassignTrait :exec
DELETE FROM character_trait_assignments WHERE character_id = $1 AND trait_id = $2;

-- name: GetTraitAssignments :many
SELECT ca.*, ct.name, ct.category, ct.description
FROM character_trait_assignments ca
JOIN character_traits ct ON ct.id = ca.trait_id
WHERE ca.character_id = $1
ORDER BY ct.name;

-- name: CreateCasting :one
INSERT INTO casting (story_id, actor_id, character_id, role_type)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListCastingForStory :many
SELECT c.*, a.name AS actor_name, ch.name AS character_name
FROM casting c
JOIN actors a ON a.id = c.actor_id
JOIN characters ch ON ch.id = c.character_id
WHERE c.story_id = $1
ORDER BY c.created_at;

-- name: ListCastingForCharacter :many
SELECT c.*, a.name AS actor_name, s.title AS story_title
FROM casting c
JOIN actors a ON a.id = c.actor_id
JOIN stories s ON s.id = c.story_id
WHERE c.character_id = $1
ORDER BY c.created_at;

-- name: ListCastingForActor :many
SELECT c.*, ch.name AS character_name, s.title AS story_title
FROM casting c
JOIN characters ch ON ch.id = c.character_id
JOIN stories s ON s.id = c.story_id
WHERE c.actor_id = $1
ORDER BY c.created_at;

-- name: CreateLocation :one
INSERT INTO locations (name, description, props)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetLocationLatest :one
SELECT * FROM latest_locations WHERE id = $1;

-- name: GetLocationAtVersion :one
SELECT * FROM locations WHERE id = $1 AND version = $2;

-- name: UpdateLocation :one
INSERT INTO locations (id, version, name, description, props)
SELECT $1, MAX(version) + 1, name, $2, CAST($3 AS jsonb)
FROM locations WHERE id = $1
GROUP BY id, name
RETURNING *;

-- name: ListLocations :many
SELECT * FROM latest_locations ORDER BY name;

-- name: CreateLore :one
INSERT INTO lore (tags, content, embedding)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SearchLoreByTags :many
SELECT * FROM lore WHERE tags && $1::text[] ORDER BY created_at DESC;

-- name: SearchLoreSimilar :many
SELECT * FROM lore
ORDER BY embedding <=> $1::vector
LIMIT $2;

-- name: ListLore :many
SELECT * FROM lore ORDER BY created_at DESC;

-- name: CreateStory :one
INSERT INTO stories (title) VALUES ($1) RETURNING *;

-- name: GetStory :one
SELECT * FROM stories WHERE id = $1;

-- name: ListStories :many
SELECT * FROM stories ORDER BY created_at DESC;

-- name: UpdateStoryCanonPins :exec
UPDATE stories SET canon_pins = $2 WHERE id = $1;

-- name: CreateNode :one
INSERT INTO nodes (story_id, beat_intent, character_refs, location_ref, pov, tone, target_words)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetNode :one
SELECT * FROM nodes WHERE id = $1;

-- name: UpdateNode :one
UPDATE nodes SET
    beat_intent = $2, character_refs = $3, location_ref = $4,
    pov = $5, tone = $6, target_words = $7,
    scene_structure = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetNodeStatus :exec
UPDATE nodes SET status = $2, updated_at = now() WHERE id = $1;

-- name: ListNodes :many
SELECT * FROM nodes WHERE story_id = $1 ORDER BY created_at;

-- name: CreateEdge :exec
INSERT INTO edges (story_id, from_node, to_node, edge_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: ListEdges :many
SELECT * FROM edges WHERE story_id = $1;

-- name: GetOutgoingEdges :many
SELECT * FROM edges WHERE from_node = $1;

-- name: GetIncomingEdges :many
SELECT * FROM edges WHERE to_node = $1;

-- name: UpdateNodeSceneStructure :exec
UPDATE nodes SET scene_structure = $2, updated_at = now() WHERE id = $1;

-- name: CreateSceneTurn :one
INSERT INTO scene_turns (node_id, turn_number, actor_ids, prompt, output, model, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListSceneTurns :many
SELECT * FROM scene_turns WHERE node_id = $1 ORDER BY turn_number;

-- name: CreateGeneration :one
INSERT INTO generations (node_id, context_hash, prompt_snapshot, output, model)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AcceptGeneration :exec
UPDATE generations SET accepted = true WHERE id = $1;

-- name: RejectOtherGenerations :exec
UPDATE generations SET accepted = false
WHERE node_id = $1 AND id != $2;

-- name: ListGenerationsForNode :many
SELECT * FROM generations WHERE node_id = $1 ORDER BY created_at DESC;

-- name: GetAcceptedGenerationForNode :one
SELECT * FROM generations WHERE node_id = $1 AND accepted = true LIMIT 1;

-- name: IsGenerationStale :one
SELECT COUNT(*) > 0 FROM generations
WHERE node_id = $1 AND context_hash != $2;

-- name: UpsertCharacterState :exec
INSERT INTO character_state (story_id, character_id, as_of_node, state)
VALUES ($1, $2, $3, $4)
ON CONFLICT (story_id, character_id, as_of_node)
DO UPDATE SET state = $4, updated_at = now();

-- name: GetCharacterState :one
SELECT * FROM character_state
WHERE story_id = $1 AND character_id = $2 AND as_of_node = $3;

-- name: GetStatesForNode :many
SELECT * FROM character_state
WHERE story_id = $1 AND as_of_node = $2;

-- name: GetStatesAtBranch :many
SELECT DISTINCT ON (cs.character_id) cs.*
FROM character_state cs
WHERE cs.story_id = $1 AND cs.as_of_node = $2
ORDER BY cs.character_id, cs.as_of_node DESC;

-- name: UpsertSceneSummary :exec
INSERT INTO story_summaries (story_id, node_id, level, content, word_count)
VALUES ($1, $2, 'scene', $3, $4)
ON CONFLICT (story_id, node_id) WHERE level = 'scene' AND node_id IS NOT NULL
DO UPDATE SET content = $3, word_count = $4;

-- name: UpsertActSummary :exec
INSERT INTO story_summaries (story_id, node_id, level, content, word_count)
VALUES ($1, NULL, 'act', $2, $3)
ON CONFLICT (story_id, level) WHERE level = 'act' AND node_id IS NULL
DO UPDATE SET content = $2, word_count = $3;

-- name: UpsertStorySummary :exec
INSERT INTO story_summaries (story_id, node_id, level, content, word_count)
VALUES ($1, NULL, 'story', $2, $3)
ON CONFLICT (story_id, level) WHERE level = 'story' AND node_id IS NULL
DO UPDATE SET content = $2, word_count = $3;

-- name: GetSceneSummary :one
SELECT * FROM story_summaries
WHERE story_id = $1 AND node_id = $2 AND level = 'scene';

-- name: GetSummaryByLevel :one
SELECT * FROM story_summaries
WHERE story_id = $1 AND level = $2
ORDER BY created_at DESC LIMIT 1;

-- name: ListSummariesByLevel :many
SELECT * FROM story_summaries
WHERE story_id = $1 AND level = $2
ORDER BY created_at DESC;

-- name: CountSummariesByLevel :one
SELECT COUNT(*) FROM story_summaries
WHERE story_id = $1 AND level = $2;

-- name: DeleteStorySummary :exec
DELETE FROM story_summaries WHERE id = $1;
