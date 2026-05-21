package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerAppliesOperationalTimeouts(t *testing.T) {
	handler := http.NewServeMux()

	server := newHTTPServer(":18080", handler)

	if server.Addr != ":18080" {
		t.Fatalf("Addr = %q, want :18080", server.Addr)
	}
	if server.Handler != handler {
		t.Fatal("Handler was not assigned")
	}
	if server.ReadHeaderTimeout != 2*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want 2s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %s, want 5s", server.ReadTimeout)
	}
	if server.WriteTimeout != 10*time.Second {
		t.Fatalf("WriteTimeout = %s, want 10s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want 60s", server.IdleTimeout)
	}
}

func TestPostgresPoolConfigAppliesPoolLimitsAndTimeouts(t *testing.T) {
	config, err := postgresPoolConfig("postgres://postgres:postgres@localhost:15432/learning_garden?sslmode=disable")
	if err != nil {
		t.Fatalf("postgresPoolConfig returned error: %v", err)
	}

	if config.MaxConns != 40 {
		t.Fatalf("MaxConns = %d, want 40", config.MaxConns)
	}
	if config.MinConns != 5 {
		t.Fatalf("MinConns = %d, want 5", config.MinConns)
	}
	if config.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("MaxConnLifetime = %s, want 30m", config.MaxConnLifetime)
	}
	if config.MaxConnIdleTime != 5*time.Minute {
		t.Fatalf("MaxConnIdleTime = %s, want 5m", config.MaxConnIdleTime)
	}
	if config.HealthCheckPeriod != 30*time.Second {
		t.Fatalf("HealthCheckPeriod = %s, want 30s", config.HealthCheckPeriod)
	}
	if config.PingTimeout != 2*time.Second {
		t.Fatalf("PingTimeout = %s, want 2s", config.PingTimeout)
	}
}

func TestRedisOptionsAppliesPoolLimitsAndTimeouts(t *testing.T) {
	options := redisOptions("localhost:16379", "secret")

	if options.Addr != "localhost:16379" {
		t.Fatalf("Addr = %q, want localhost:16379", options.Addr)
	}
	if options.Password != "secret" {
		t.Fatalf("Password = %q, want secret", options.Password)
	}
	if options.DialTimeout != 2*time.Second {
		t.Fatalf("DialTimeout = %s, want 2s", options.DialTimeout)
	}
	if options.ReadTimeout != 1*time.Second {
		t.Fatalf("ReadTimeout = %s, want 1s", options.ReadTimeout)
	}
	if options.WriteTimeout != 1*time.Second {
		t.Fatalf("WriteTimeout = %s, want 1s", options.WriteTimeout)
	}
	if options.PoolTimeout != 2*time.Second {
		t.Fatalf("PoolTimeout = %s, want 2s", options.PoolTimeout)
	}
	if options.PoolSize != 100 {
		t.Fatalf("PoolSize = %d, want 100", options.PoolSize)
	}
	if options.MinIdleConns != 10 {
		t.Fatalf("MinIdleConns = %d, want 10", options.MinIdleConns)
	}
	if !options.ContextTimeoutEnabled {
		t.Fatal("ContextTimeoutEnabled = false, want true")
	}
	if options.ConnMaxIdleTime != 5*time.Minute {
		t.Fatalf("ConnMaxIdleTime = %s, want 5m", options.ConnMaxIdleTime)
	}
	if options.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("ConnMaxLifetime = %s, want 30m", options.ConnMaxLifetime)
	}
}
