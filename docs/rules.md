# Migration Rules

Tracking table: `_migrations` (Flyway-compatible)

Schema (`_migrations`):

| Column          | Type      | Notes                    |
|-----------------|-----------|--------------------------|
| `version`       | `int` PK  | Migration number         |
| `filename`      | `text`    | `.sql` file name         |
| `description`   | `text`    | Human-readable summary   |
| `migration_type`| `text`    | `sql` (default)          |
| `checksum`      | `text`    | MD5 hash of file content |
| `installed_by`  | `text`    | `current_user` (default) |
| `installed_on`  | `timestamptz` | `now()` (default)     |
| `execution_time`| `int`     | Seconds (default 0)      |
| `success`       | `bool`    | `true` (default)         |

## File naming

```
{VERSION}_{DESCRIPTION}.sql
```

- `VERSION` = zero-padded integer, e.g. `001`, `002`, `010`, `101`
- `DESCRIPTION` = snake_case summary, e.g. `add_character_voice_samples`

Examples: `002_add_prop_table.sql`, `015_fix_character_state_index.sql`

## Rules

1. **Append-only.** Never modify an existing migration file. Write a new migration to:
   - Add/alter/drop columns
   - Add/remove indexes
   - Seed or transform data
   - Fix past mistakes

2. **One transaction per file.** The runner wraps each file in a transaction automatically. If your migration needs to run outside a transaction (e.g. `CREATE INDEX CONCURRENTLY`, `ALTER TYPE`), split it into its own file and note it.

3. **Idempotent where possible.** Use `IF NOT EXISTS` / `IF EXISTS` / `CREATE OR REPLACE` so re-runs don't error.

4. **Checksum enforced.** The runner stores an MD5 of each applied file. If a file changes after being applied, the runner will fail on next startup. To change a past migration, write a new file that reverts + reapplies.

5. **No editing applied files.** The `001_init.sql` file is the baseline and must never change. All schema changes go in new numbered files.

6. **Test down?** We don't do rollback migrations. If a migration breaks things, write a new migration that reverses it. Production data recovery is a separate process.

## Workflow

```bash
# Create a new migration
touch migrations/003_add_prop_table.sql
# Edit it, then restart the server (runner auto-applies on startup)

# Check pending migrations
# Run the server — it logs each applied migration
```

## Migration runner

The Go runner in `internal/migrate/runner.go`:
- Runs on server startup (only if DB is connected)
- Creates `_migrations` tracking table
- Scans `migrations/*.sql` files sorted by version prefix
- Applies any unapplied files in a transaction
- Records version + checksum in `_migrations`
