package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/municipios"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("municipios import failed", "error", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string) error {
	_ = godotenv.Load()

	flags := flag.NewFlagSet("import-municipios", flag.ContinueOnError)
	uf := flags.String("uf", "", "import only one UF, for example AL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	*uf = strings.ToUpper(strings.TrimSpace(*uf))
	if *uf != "" && (len(*uf) != 2 || (*uf)[0] < 'A' || (*uf)[0] > 'Z' || (*uf)[1] < 'A' || (*uf)[1] > 'Z') {
		return errors.New("uf must contain exactly two letters")
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(parent, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := municipios.NewIBGEClient(httpClient, os.Getenv("IBGE_BASE_URL"))
	items, err := client.List(ctx, *uf)
	if err != nil {
		return fmt.Errorf("fetch IBGE municipios: %w", err)
	}

	if err := municipios.NewStore(pool).Upsert(ctx, items); err != nil {
		return err
	}

	slog.Info("municipios imported", "count", len(items), "uf", *uf)
	return nil
}
