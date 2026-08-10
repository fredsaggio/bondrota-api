package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/server"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = godotenv.Load()
	if err := Run(context.Background(), os.Getenv); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func Run(ctx context.Context, getEnv func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	dbURL := getEnv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := getEnv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	baseCity := strings.TrimSpace(getEnv("BASE_CITY"))
	if baseCity == "" {
		return fmt.Errorf("BASE_CITY is required")
	}

	appTimezone := strings.TrimSpace(getEnv("APP_TIMEZONE"))
	if appTimezone == "" {
		return fmt.Errorf("APP_TIMEZONE is required")
	}
	appLocation, err := time.LoadLocation(appTimezone)
	if err != nil {
		return fmt.Errorf("invalid APP_TIMEZONE: %w", err)
	}

	hasher := crypto.NewBcryptHasher(crypto.DefaultCost)

	authSvc := auth.NewAuthService(hasher, jwtSecret)

	port := getEnv("PORT")
	if port == "" {
		port = "8080"
	}

	allowedOrigins := getEnv("ALLOWED_ORIGINS")

	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(allowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database")
	}
	defer pool.Close()

	slog.Info("database connected")

	storageConfig := storage.SupabaseConfig{
		URL:        getEnv("SUPABASE_URL"),
		ServiceKey: getEnv("SUPABASE_SERVICE_KEY"),
	}

	handlers, rotaDinamicaWorker := buildHandlers(pool, authSvc, storageConfig, getEnv("OSRM_BASE_URL"), appLocation)
	srv := server.NewServer(handlers, authSvc, server.Config{BaseCity: baseCity})
	apiRouter := chi.NewRouter()
	srv.RegisterRoutes(apiRouter)
	r.Mount("/api/v1", apiRouter)

	httpSrv := &http.Server{Addr: ":" + port, Handler: r}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("server started", "port", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		slog.Info("dynamic route worker started")
		rotaDinamicaWorker.Run(ctx)
		slog.Info("dynamic route worker stopped")
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		slog.Info("shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}

		slog.Info("server stopped")
		return nil
	})

	return g.Wait()
}
