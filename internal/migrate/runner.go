package migrate

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct {
	pool *pgxpool.Pool
	dir  string
}

func New(pool *pgxpool.Pool, migrationsDir string) *Runner {
	return &Runner{pool: pool, dir: migrationsDir}
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("migrate: ensure table: %w", err)
	}

	files, err := os.ReadDir(r.dir)
	if err != nil {
		return fmt.Errorf("migrate: read dir: %w", err)
	}

	type migration struct {
		version  int
		filename string
	}

	var migrations []migration
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(f.Name(), "_", 2)
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		migrations = append(migrations, migration{version: v, filename: f.Name()})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for _, m := range migrations {
		applied, err := r.isApplied(ctx, m.version)
		if err != nil {
			return fmt.Errorf("migrate: check %s: %w", m.filename, err)
		}
		if applied {
			continue
		}

		slog.Info("applying migration", "file", m.filename)
		if err := r.apply(ctx, m.filename); err != nil {
			return fmt.Errorf("migrate: apply %s: %w", m.filename, err)
		}
		slog.Info("applied migration", "file", m.filename)
	}

	return nil
}

func (r *Runner) ensureTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _migrations (
			version         int PRIMARY KEY,
			filename        text NOT NULL,
			description     text NOT NULL DEFAULT '',
			migration_type  text NOT NULL DEFAULT 'sql',
			checksum        text NOT NULL DEFAULT '',
			installed_by    text NOT NULL DEFAULT current_user,
			installed_on    timestamptz NOT NULL DEFAULT now(),
			execution_time  int NOT NULL DEFAULT 0,
			success         bool NOT NULL DEFAULT true
		)
	`)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO _migrations (version, filename, description, checksum)
		SELECT 1, '001_init.sql',
		'Initial schema: characters, locations, lore, stories, nodes, edges, generations, character_state',
		'entrypoint'
		WHERE NOT EXISTS (SELECT 1 FROM _migrations WHERE version = 1)
	`)
	return err
}

func (r *Runner) isApplied(ctx context.Context, version int) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM _migrations WHERE version = $1 AND success = true`, version).Scan(&count)
	return count > 0, err
}

func (r *Runner) apply(ctx context.Context, filename string) error {
	path := filepath.Join(r.dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	checksum := fmt.Sprintf("%x", md5.Sum(content))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return err
	}

	parts := strings.SplitN(filename, "_", 2)
	version, _ := strconv.Atoi(parts[0])

	_, err = tx.Exec(ctx,
		`INSERT INTO _migrations (version, filename, checksum) VALUES ($1, $2, $3)`,
		version, filename, checksum)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Runner) Pending(ctx context.Context) ([]string, error) {
	return nil, nil
}
