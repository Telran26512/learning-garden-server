package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/authrepo"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
	"github.com/Telran26512/learning-garden-server/services/api/internal/contentrepo"
	"github.com/Telran26512/learning-garden-server/services/api/internal/httpapi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 10 * time.Second

	httpReadHeaderTimeout = 2 * time.Second
	httpReadTimeout       = 5 * time.Second
	httpWriteTimeout      = 10 * time.Second
	httpIdleTimeout       = 60 * time.Second

	postgresMaxConns          int32 = 40
	postgresMinConns          int32 = 5
	postgresMaxConnLifetime         = 30 * time.Minute
	postgresMaxConnIdleTime         = 5 * time.Minute
	postgresHealthCheckPeriod       = 30 * time.Second
	postgresPingTimeout             = 2 * time.Second

	redisDialTimeout     = 2 * time.Second
	redisReadTimeout     = 1 * time.Second
	redisWriteTimeout    = 1 * time.Second
	redisPoolTimeout     = 2 * time.Second
	redisPoolSize        = 100
	redisMinIdleConns    = 10
	redisConnMaxIdleTime = 5 * time.Minute
	redisConnMaxLifetime = 30 * time.Minute
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupCtx, cancelStartup := context.WithTimeout(ctx, startupTimeout)
	defer cancelStartup()

	dbConfig, err := postgresPoolConfig(env("DATABASE_URL", "postgres://postgres:postgres@localhost:15432/learning_garden?sslmode=disable"))
	if err != nil {
		log.Fatalf("configure postgres: %v", err)
	}
	db, err := pgxpool.NewWithConfig(startupCtx, dbConfig)
	if err != nil {
		log.Fatalf("create postgres pool: %v", err)
	}
	if err := db.Ping(startupCtx); err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	redisClient := redis.NewClient(redisOptions(env("REDIS_ADDR", "localhost:16379"), os.Getenv("REDIS_PASSWORD")))
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	authService := auth.NewService(
		authrepo.NewPostgresUserStore(db),
		authrepo.NewRedisRefreshStore(redisClient),
		auth.Config{
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
			JWTSecret:       env("JWT_SECRET", "dev-only-change-me"),
		},
	)

	router := httpapi.NewRouter(httpapi.NewRouterConfig{
		Auth:           authService,
		Content:        content.NewService(contentrepo.NewPostgresContentStore(db)),
		AllowedOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000,http://127.0.0.1:3001")),
		CookieSecure:   env("COOKIE_SECURE", "false") == "true",
		DBStats: func() httpapi.DBPoolStats {
			return db.Stat()
		},
	})

	addr := env("HTTP_ADDR", ":18080")
	server := newHTTPServer(addr, router)
	serverErr := make(chan error, 1)

	go func() {
		log.Printf("synapse api listening on %s", addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server stopped: %v", err)
		}
		return
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
	}
}

func postgresPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = postgresMaxConns
	config.MinConns = postgresMinConns
	config.MaxConnLifetime = postgresMaxConnLifetime
	config.MaxConnIdleTime = postgresMaxConnIdleTime
	config.HealthCheckPeriod = postgresHealthCheckPeriod
	config.PingTimeout = postgresPingTimeout
	return config, nil
}

func redisOptions(addr string, password string) *redis.Options {
	return &redis.Options{
		Addr:                  addr,
		Password:              password,
		DB:                    0,
		DialTimeout:           redisDialTimeout,
		ReadTimeout:           redisReadTimeout,
		WriteTimeout:          redisWriteTimeout,
		ContextTimeoutEnabled: true,
		PoolSize:              redisPoolSize,
		PoolTimeout:           redisPoolTimeout,
		MinIdleConns:          redisMinIdleConns,
		ConnMaxIdleTime:       redisConnMaxIdleTime,
		ConnMaxLifetime:       redisConnMaxLifetime,
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
