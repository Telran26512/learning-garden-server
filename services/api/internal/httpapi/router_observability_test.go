package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestIDMiddlewareSetsResponseHeader(t *testing.T) {
	handler := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestIDFromContext(r.Context()) == "" {
			t.Fatal("request id missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, "client-request-id")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != "client-request-id" {
		t.Fatalf("%s = %q, want client-request-id", requestIDHeader, got)
	}
}

func TestRecoveryMiddlewareReturnsInternalErrorEnvelope(t *testing.T) {
	handler := recoveryMiddleware(log.New(&bytes.Buffer{}, "", 0))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"INTERNAL"`) {
		t.Fatalf("body missing INTERNAL error: %s", rec.Body.String())
	}
}

func TestAccessLogMiddlewareWritesRequestSummary(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	handler := accessLogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req = req.WithContext(contextWithRequestID(req.Context(), "req-123"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	line := output.String()
	for _, want := range []string{"request completed", "request_id=req-123", "method=POST", "path=/auth/login", "status=202"} {
		if !strings.Contains(line, want) {
			t.Fatalf("access log %q missing %q", line, want)
		}
	}
}

func TestRequestTimeoutMiddlewareAddsDeadline(t *testing.T) {
	handler := requestTimeoutMiddleware(3 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("request context has no deadline")
		}
		if time.Until(deadline) <= 0 || time.Until(deadline) > 3*time.Second {
			t.Fatalf("deadline = %s, want within 3s", deadline)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestRouterExposesPrometheusHTTPRuntimeAndDBMetrics(t *testing.T) {
	router := NewRouter(NewRouterConfig{
		DBStats: func() DBPoolStats {
			return staticDBPoolStats{}
		},
	})

	health := doJSON(t, router, http.MethodGet, "/healthz", nil, nil)
	if health.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, body = %s", health.Code, health.Body.String())
	}

	metrics := doJSON(t, router, http.MethodGet, "/metrics", nil, nil)
	if metrics.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, body = %s", metrics.Code, metrics.Body.String())
	}
	body := metrics.Body.String()
	for _, want := range []string{
		`learning_garden_http_requests_total{method="GET",route="/healthz",status="200"} 1`,
		"learning_garden_http_request_duration_seconds_bucket",
		"learning_garden_http_requests_in_flight",
		"go_goroutines",
		"learning_garden_db_pool_acquired_conns 2",
		"learning_garden_db_pool_idle_conns 3",
		"learning_garden_db_pool_total_conns 5",
		"learning_garden_db_pool_max_conns 40",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}

type staticDBPoolStats struct{}

func (staticDBPoolStats) AcquiredConns() int32        { return 2 }
func (staticDBPoolStats) IdleConns() int32            { return 3 }
func (staticDBPoolStats) TotalConns() int32           { return 5 }
func (staticDBPoolStats) MaxConns() int32             { return 40 }
func (staticDBPoolStats) AcquireCount() int64         { return 10 }
func (staticDBPoolStats) CanceledAcquireCount() int64 { return 1 }
func (staticDBPoolStats) EmptyAcquireCount() int64    { return 4 }
