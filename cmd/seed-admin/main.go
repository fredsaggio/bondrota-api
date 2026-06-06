package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("seed admin failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	_ = godotenv.Load(".env")
	_ = godotenv.Overload(".env.prod")

	dbURL := strings.TrimSpace(os.Getenv("PROD_DATABASE_URL"))
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")

	if dbURL == "" {
		return errors.New("PROD_DATABASE_URL is required")
	}
	if email == "" {
		return errors.New("ADMIN_EMAIL is required")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("ADMIN_PASSWORD is required")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	exists, err := adminExists(ctx, pool, email)
	if err != nil {
		return err
	}
	if exists {
		slog.Info("admin already exists", "email", email)
		return nil
	}

	hasher := crypto.NewBcryptHasher(crypto.DefaultCost)
	hashedPassword, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	const q = `
		INSERT INTO administrador (email, senha)
		VALUES (@email, @senha)
	`
	if _, err := pool.Exec(ctx, q, pgx.StrictNamedArgs{
		"email": email,
		"senha": hashedPassword,
	}); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	slog.Info("admin created", "email", email)
	return nil
}

func adminExists(ctx context.Context, querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, email string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM administrador WHERE email = @email)`

	var exists bool
	if err := querier.QueryRow(ctx, q, pgx.StrictNamedArgs{"email": email}).Scan(&exists); err != nil {
		return false, fmt.Errorf("check admin exists: %w", err)
	}

	return exists, nil
}
