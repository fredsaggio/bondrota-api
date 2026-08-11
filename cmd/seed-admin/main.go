package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
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

const (
	targetLocal = "local"
	targetProd  = "prod"
)

// databaseEnvFor devolve a variavel que guarda a URL do banco de cada alvo. O alvo
// padrao e o local: producao exige `-target=prod` explicito, para que rodar o
// comando durante o desenvolvimento nunca crie um administrador em producao.
func databaseEnvFor(target string) (string, error) {
	switch target {
	case targetLocal:
		return "DATABASE_URL", nil
	case targetProd:
		return "PROD_DATABASE_URL", nil
	default:
		return "", fmt.Errorf("invalid -target %q: use %s or %s", target, targetLocal, targetProd)
	}
}

// safeHost extrai apenas host:porta da URL de conexao, para registrar o alvo sem
// expor usuario e senha no log.
func safeHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "desconhecido"
	}
	return parsed.Host
}

func run(ctx context.Context) error {
	target := flag.String("target", targetLocal, "banco alvo: local (DATABASE_URL) ou prod (PROD_DATABASE_URL)")
	flag.Parse()

	dbEnv, err := databaseEnvFor(*target)
	if err != nil {
		return err
	}

	_ = godotenv.Load(".env")
	if *target == targetProd {
		// .env.prod so e carregado no alvo de producao. Carregar sempre faria
		// PROD_DATABASE_URL vazar para execucoes locais.
		_ = godotenv.Overload(".env.prod")
	}

	dbURL := strings.TrimSpace(os.Getenv(dbEnv))
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")

	if dbURL == "" {
		return fmt.Errorf("%s is required for -target=%s", dbEnv, *target)
	}
	if email == "" {
		return errors.New("ADMIN_EMAIL is required")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("ADMIN_PASSWORD is required")
	}

	slog.Info("seeding admin", "target", *target, "host", safeHost(dbURL), "email", email)

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
