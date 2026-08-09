//go:build integration

package repositories

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/db/dbtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testDatabase, err := dbtest.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup integration database: %v\n", err)
		os.Exit(1)
	}
	testPool = testDatabase.Pool()

	code := m.Run()
	if err := testDatabase.Close(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "terminate integration database: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
