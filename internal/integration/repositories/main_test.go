//go:build integration

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("bondrota_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start integration database: %v\n", err)
		os.Exit(1)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "get integration database URL: %v\n", err)
		os.Exit(1)
	}

	if err := applyMigrations(ctx, dsn); err != nil {
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "apply integration migrations: %v\n", err)
		os.Exit(1)
	}

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "connect integration database: %v\n", err)
		os.Exit(1)
	}
	if err := testPool.Ping(ctx); err != nil {
		testPool.Close()
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "ping integration database: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	testPool.Close()
	if err := container.Terminate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "terminate integration database: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func applyMigrations(ctx context.Context, dsn string) error {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer database.Close()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve migration directory")
	}
	migrationDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "db", "migrations")

	provider, err := goose.NewProvider(goose.DialectPostgres, database, os.DirFS(migrationDir))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
