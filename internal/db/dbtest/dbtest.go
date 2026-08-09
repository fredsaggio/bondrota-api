package dbtest

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/db/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type TestDB struct {
	container *postgres.PostgresContainer
	pool      *pgxpool.Pool
}

func New(ctx context.Context) (_ *TestDB, err error) {
	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("bondrota_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start PostgreSQL container: %w", err)
	}

	testDB := &TestDB{container: container}
	defer func() {
		if err != nil {
			_ = testDB.Close(context.Background())
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection string: %w", err)
	}
	if err := applyMigrations(ctx, dsn); err != nil {
		return nil, err
	}

	testDB.pool, err = db.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL container: %w", err)
	}
	return testDB, nil
}

func (tdb *TestDB) Pool() *pgxpool.Pool {
	return tdb.pool
}

func (tdb *TestDB) Close(ctx context.Context) error {
	if tdb.pool != nil {
		tdb.pool.Close()
		tdb.pool = nil
	}
	if tdb.container == nil {
		return nil
	}

	err := tdb.container.Terminate(ctx)
	tdb.container = nil
	if err != nil {
		return fmt.Errorf("terminate PostgreSQL container: %w", err)
	}
	return nil
}

func applyMigrations(ctx context.Context, dsn string) error {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer database.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	defer provider.Close()

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
