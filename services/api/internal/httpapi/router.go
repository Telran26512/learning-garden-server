package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
	"github.com/go-chi/chi/v5"
)

const (
	refreshCookieName     = "synapse_refresh"
	defaultRequestTimeout = 3 * time.Second
)

type NewRouterConfig struct {
	Auth           *auth.Service
	Content        *content.Service
	AllowedOrigins []string
	CookieSecure   bool
	DBStats        func() DBPoolStats
	Logger         *log.Logger
	RequestTimeout time.Duration
}

func NewRouter(config NewRouterConfig) http.Handler {
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}
	requestTimeout := config.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = defaultRequestTimeout
	}
	metrics := newRouterMetrics(config.DBStats)

	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Use(accessLogMiddleware(logger))
	r.Use(metrics.middleware)
	r.Use(recoveryMiddleware(logger))
	r.Use(requestTimeoutMiddleware(requestTimeout))
	if len(config.AllowedOrigins) > 0 {
		r.Use(corsMiddleware(config.AllowedOrigins))
	}

	handler := authHandler{service: config.Auth, cookieSecure: config.CookieSecure}
	contentHandler := contentHandler{auth: config.Auth, content: config.Content}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	r.Handle("/metrics", metrics.handler())
	r.Post("/auth/register", handler.register)
	r.Post("/auth/login", handler.login)
	r.Post("/auth/refresh", handler.refresh)
	r.Post("/auth/logout", handler.logout)
	r.Get("/auth/me", handler.me)
	r.Get("/auth/github", handler.github)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/content", contentHandler.listContent)
		r.Post("/content", contentHandler.createContent)
		r.Get("/content/{id}", contentHandler.getContent)
		r.Patch("/content/{id}", contentHandler.updateContent)
		r.Delete("/content/{id}", contentHandler.deleteContent)
		r.Post("/content/{id}/relations", contentHandler.addRelation)
		r.Get("/content/{id}/backlinks", contentHandler.backlinks)
		r.Get("/users/{handle}", contentHandler.publicProfile)
		r.Get("/users/{handle}/content", contentHandler.userContent)
		r.Get("/portfolio/{handle}", contentHandler.portfolio)
		r.Get("/graph", contentHandler.graph)
		r.Get("/community/feed", contentHandler.communityFeed)
	})

	return r
}

type authHandler struct {
	service      *auth.Service
	cookieSecure bool
}

func (h authHandler) register(w http.ResponseWriter, r *http.Request) {
	var input auth.RegisterInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Register(r.Context(), input)
	if err != nil {
		writeAppError(w, authAppError(err))
		return
	}
	h.setRefreshCookie(w, session.RefreshToken)
	writeJSON(w, http.StatusCreated, session)
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var input auth.LoginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeAppError(w, authAppError(err))
		return
	}
	h.setRefreshCookie(w, session.RefreshToken)
	writeJSON(w, http.StatusOK, session)
}

func (h authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeAppError(w, appError(http.StatusUnauthorized, "UNAUTHORIZED", "refresh cookie is missing"))
		return
	}
	session, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeAppError(w, authAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeAppError(w, appError(http.StatusUnauthorized, "UNAUTHORIZED", "access token is missing"))
		return
	}
	user, err := h.service.AuthenticateAccessToken(r.Context(), token)
	if err != nil {
		writeAppError(w, authAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h authHandler) github(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, appError(http.StatusNotImplemented, "OAUTH_NOT_CONFIGURED", "GitHub OAuth is not configured"))
}

func (h authHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().UTC().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
	})
}

func bearerToken(value string) string {
	prefix := "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAppError(w, appError(http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":  data,
		"error": nil,
		"meta":  map[string]any{},
	})
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
