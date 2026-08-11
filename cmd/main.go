package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/retencao"
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

	planningCronSecret := strings.TrimSpace(getEnv("PLANNING_CRON_SECRET"))
	if len(planningCronSecret) < 32 {
		return fmt.Errorf("PLANNING_CRON_SECRET must contain at least 32 characters")
	}

	hasher := crypto.NewBcryptHasher(crypto.DefaultCost)

	authSvc := auth.NewAuthService(hasher, jwtSecret)

	port := getEnv("PORT")
	if port == "" {
		port = "8080"
	}

	allowedOrigins, err := parseAllowedOrigins(getEnv("ALLOWED_ORIGINS"))
	if err != nil {
		return err
	}
	adminCookieConfig, err := loadAdminCookieConfig(getEnv, allowedOrigins)
	if err != nil {
		return err
	}
	loginRateLimitConfig, err := loadLoginRateLimitConfig(getEnv)
	if err != nil {
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", admin.SessionModeHeader},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(auth.ProtectCookieRequests(allowedOrigins, adminCookieConfig.Name))

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

	retencaoMeses, err := positiveIntEnv(getEnv, "RETENTION_MONTHS", retencao.MesesPadrao)
	if err != nil {
		return err
	}
	retencaoLote, err := positiveIntEnv(getEnv, "RETENTION_BATCH_LIMIT", retencao.LoteMaximoPadrao)
	if err != nil {
		return err
	}
	retencaoConfig := retencao.Config{
		Meses:      retencaoMeses,
		LoteMaximo: retencaoLote,
		Location:   appLocation,
	}

	handlers, rotaDinamicaWorker := buildHandlers(pool, authSvc, adminCookieConfig, storageConfig, getEnv("OSRM_BASE_URL"), appLocation, retencaoConfig)
	srv := server.NewServer(handlers, authSvc, server.Config{
		BaseCity:           baseCity,
		TimeZone:           appTimezone,
		PlanningCronSecret: planningCronSecret,
		AdminCookieName:    adminCookieConfig.Name,
		LoginRateLimit:     loginRateLimitConfig,
	})
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

func parseAllowedOrigins(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "http://localhost:3000"
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		origin := strings.TrimRight(strings.TrimSpace(value), "/")
		if origin == "" {
			continue
		}
		if origin == "*" {
			return nil, fmt.Errorf("ALLOWED_ORIGINS cannot contain * when credentials are enabled")
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("ALLOWED_ORIGINS must contain at least one origin")
	}
	return origins, nil
}

func loadAdminCookieConfig(getEnv func(string) string, allowedOrigins []string) (admin.SessionCookieConfig, error) {
	allOriginsAreLocal := true
	for _, origin := range allowedOrigins {
		if !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
			allOriginsAreLocal = false
			break
		}
	}
	secure := !allOriginsAreLocal
	if raw := strings.TrimSpace(getEnv("AUTH_COOKIE_SECURE")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return admin.SessionCookieConfig{}, fmt.Errorf("invalid AUTH_COOKIE_SECURE: %w", err)
		}
		secure = parsed
	}

	sameSite := http.SameSiteLaxMode
	switch strings.ToLower(strings.TrimSpace(getEnv("AUTH_COOKIE_SAME_SITE"))) {
	case "", "lax":
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	default:
		return admin.SessionCookieConfig{}, fmt.Errorf("AUTH_COOKIE_SAME_SITE must be lax, strict or none")
	}
	if sameSite == http.SameSiteNoneMode && !secure {
		return admin.SessionCookieConfig{}, fmt.Errorf("AUTH_COOKIE_SECURE must be true when AUTH_COOKIE_SAME_SITE=none")
	}

	name := strings.TrimSpace(getEnv("AUTH_COOKIE_NAME"))
	if name == "" {
		name = admin.DefaultSessionCookieName
	}
	return admin.SessionCookieConfig{
		Name:     name,
		Path:     "/api/v1",
		Domain:   strings.TrimSpace(getEnv("AUTH_COOKIE_DOMAIN")),
		Secure:   secure,
		SameSite: sameSite,
		TTL:      auth.TokenTTL,
	}, nil
}

func loadLoginRateLimitConfig(getEnv func(string) string) (server.LoginRateLimitConfig, error) {
	perIP, err := positiveIntEnv(getEnv, "LOGIN_RATE_LIMIT_PER_IP", 20)
	if err != nil {
		return server.LoginRateLimitConfig{}, err
	}
	perIdentity, err := positiveIntEnv(getEnv, "LOGIN_RATE_LIMIT_PER_IDENTITY", 5)
	if err != nil {
		return server.LoginRateLimitConfig{}, err
	}

	window := time.Minute
	if raw := strings.TrimSpace(getEnv("LOGIN_RATE_LIMIT_WINDOW")); raw != "" {
		window, err = time.ParseDuration(raw)
		if err != nil || window <= 0 {
			return server.LoginRateLimitConfig{}, fmt.Errorf("LOGIN_RATE_LIMIT_WINDOW must be a positive duration")
		}
	}

	trustProxyHeaders := false
	if raw := strings.TrimSpace(getEnv("LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS")); raw != "" {
		trustProxyHeaders, err = strconv.ParseBool(raw)
		if err != nil {
			return server.LoginRateLimitConfig{}, fmt.Errorf("invalid LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS: %w", err)
		}
	}

	return server.LoginRateLimitConfig{
		RequestsPerIP:       perIP,
		RequestsPerIdentity: perIdentity,
		Window:              window,
		TrustProxyHeaders:   trustProxyHeaders,
	}, nil
}

func positiveIntEnv(getEnv func(string) string, name string, fallback int) (int, error) {
	raw := strings.TrimSpace(getEnv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
