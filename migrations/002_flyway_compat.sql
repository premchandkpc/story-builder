-- Flyway-compatible migration tracking table
-- Adds standard Flyway columns to the _migrations table created by the Go runner.

ALTER TABLE _migrations ADD COLUMN IF NOT EXISTS installed_by text NOT NULL DEFAULT current_user;
ALTER TABLE _migrations ADD COLUMN IF NOT EXISTS execution_time int NOT NULL DEFAULT 0;
ALTER TABLE _migrations ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';
ALTER TABLE _migrations ADD COLUMN IF NOT EXISTS migration_type text NOT NULL DEFAULT 'sql';

-- Update existing record for migration 001 to fill Flyway fields
UPDATE _migrations
SET description = 'Initial schema: characters, locations, lore, stories, nodes, edges, generations, character_state',
    migration_type = 'sql',
    execution_time = 0
WHERE version = 1;
