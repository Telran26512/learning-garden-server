package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/httpapi"
	"github.com/Telran26512/learning-garden-server/services/api/internal/repo"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://postgres:postgres@localhost:15432/learning_garden?sslmode=disable"))
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:16379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	authService := auth.NewService(
		repo.NewPostgresUserStore(db),
		repo.NewRedisRefreshStore(redisClient),
		auth.Config{
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
			JWTSecret:       env("JWT_SECRET", "dev-only-change-me"),
		},
	)

	router := httpapi.NewRouter(httpapi.NewRouterConfig{
		Auth:           authService,
		AllowedOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000,http://127.0.0.1:3001")),
		CookieSecure:   env("COOKIE_SECURE", "false") == "true",
	})

	addr := env("HTTP_ADDR", ":18080")
	log.Printf("synapse api listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
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
