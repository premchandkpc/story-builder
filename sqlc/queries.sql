-- name: CreateCharacter :one
INSERT INTO characters (name, traits, voice_samples, relationships)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCharacterLatest :one
SELECT * FROM latest_characters WHERE id = $1;

-- name: GetCharacterAtVersion :one
SELECT * FROM characters WHERE id = $1 AND version = $2;

-- name: UpdateCharacter :one
INSERT INTO characters (id, version, name, traits, voice_samples, relationships)
SELECT c.id, MAX(c.version) + 1, c.name, CAST($2 AS jsonb), $3, CAST($4 AS jsonb)
FROM characters c WHERE c.id = $1
GROUP BY c.id, c.name
RETURNING *;

-- name: ListCharacters :many
SELECT * FROM latest_characters ORDER BY name;

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
    pov = $5, tone = $6, target_words = $7, updated_at = now()
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
