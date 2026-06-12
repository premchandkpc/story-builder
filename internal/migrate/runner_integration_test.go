//go:build integration

package migrate

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	postgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRunnerAppliesMigrationsAgainstPostgres(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("storybuilder"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			t.Fatalf("terminate postgres container: %v", terminateErr)
		}
	}()

	connString, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		t.Fatalf("create vector extension: %v", err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}

	runner := New(pool, filepath.Join(filepath.Dir(filename), "..", "..", "migrations"))
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM _migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least one migration, got %d", count)
	}
}
