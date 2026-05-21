package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
	"github.com/go-chi/chi/v5"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type contentHandler struct {
	auth    *auth.Service
	content *content.Service
}

func (h contentHandler) listContent(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		writeAppError(w, appError(http.StatusServiceUnavailable, "CONTENT_UNAVAILABLE", "content service is not configured"))
		return
	}
	filter := content.ListFilter{
		Kind:  content.Kind(strings.TrimSpace(r.URL.Query().Get("kind"))),
		Limit: limitQuery(r),
	}
	items, err := h.content.ListPublic(r.Context(), filter)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": len(items),
	})
}

func (h contentHandler) createContent(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input content.CreateItemInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := h.content.Create(r.Context(), user.ID, input)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h contentHandler) getContent(w http.ResponseWriter, r *http.Request) {
	viewer, ok := h.optionalUser(w, r)
	if !ok {
		return
	}
	viewerID := ""
	if viewer != nil {
		viewerID = viewer.ID
	}
	item, err := h.content.FindVisible(r.Context(), chi.URLParam(r, "id"), viewerID)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h contentHandler) updateContent(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input content.UpdateItemInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := h.content.Update(r.Context(), chi.URLParam(r, "id"), user.ID, input)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h contentHandler) deleteContent(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.content.Delete(r.Context(), chi.URLParam(r, "id"), user.ID); err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h contentHandler) addRelation(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input struct {
		TargetID string `json:"targetId"`
		Type     string `json:"type"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	relation, err := h.content.AddRelation(r.Context(), chi.URLParam(r, "id"), user.ID, input.TargetID, input.Type)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusCreated, relation)
}

func (h contentHandler) backlinks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.content.Backlinks(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": rows,
		"total": len(rows),
	})
}

func (h contentHandler) publicProfile(w http.ResponseWriter, r *http.Request) {
	profile, ok := h.profileForHandle(w, r, chi.URLParam(r, "handle"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h contentHandler) userContent(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.FindUserByHandle(r.Context(), chi.URLParam(r, "handle"))
	if err != nil {
		writeAppError(w, appError(http.StatusNotFound, "NOT_FOUND", "user not found"))
		return
	}
	items, err := h.content.ListPublic(r.Context(), content.ListFilter{
		OwnerID: user.ID,
		Kind:    content.Kind(strings.TrimSpace(r.URL.Query().Get("kind"))),
		Limit:   limitQuery(r),
	})
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": len(items),
	})
}

func (h contentHandler) portfolio(w http.ResponseWriter, r *http.Request) {
	profile, ok := h.profileForHandle(w, r, chi.URLParam(r, "handle"))
	if !ok {
		return
	}
	portfolio, err := h.content.Portfolio(r.Context(), profile, limitQuery(r))
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, portfolio)
}

func (h contentHandler) graph(w http.ResponseWriter, r *http.Request) {
	filter := content.ListFilter{Limit: limitQuery(r)}
	if handle := strings.TrimSpace(r.URL.Query().Get("handle")); handle != "" {
		user, err := h.auth.FindUserByHandle(r.Context(), handle)
		if err != nil {
			writeAppError(w, appError(http.StatusNotFound, "NOT_FOUND", "user not found"))
			return
		}
		filter.OwnerID = user.ID
	}
	graph, err := h.content.Graph(r.Context(), filter)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (h contentHandler) communityFeed(w http.ResponseWriter, r *http.Request) {
	rows, err := h.content.CommunityFeed(r.Context(), limitQuery(r))
	if err != nil {
		writeAppError(w, contentAppError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": rows,
		"total": len(rows),
	})
}

func (h contentHandler) profileForHandle(w http.ResponseWriter, r *http.Request, handle string) (content.PublicProfile, bool) {
	user, err := h.auth.FindUserByHandle(r.Context(), handle)
	if err != nil {
		writeAppError(w, appError(http.StatusNotFound, "NOT_FOUND", "user not found"))
		return content.PublicProfile{}, false
	}
	stats, err := h.content.Stats(r.Context(), user.ID)
	if err != nil {
		writeAppError(w, contentAppError(err))
		return content.PublicProfile{}, false
	}
	return content.PublicProfile{
		ID:          user.ID,
		Handle:      user.Handle,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		Stats:       stats,
	}, true
}

func (h contentHandler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if h.auth == nil || h.content == nil {
		writeAppError(w, appError(http.StatusServiceUnavailable, "CONTENT_UNAVAILABLE", "content service is not configured"))
		return auth.User{}, false
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeAppError(w, appError(http.StatusUnauthorized, "UNAUTHORIZED", "access token is missing"))
		return auth.User{}, false
	}
	user, err := h.auth.AuthenticateAccessToken(r.Context(), token)
	if err != nil {
		writeAppError(w, authAppError(err))
		return auth.User{}, false
	}
	return user, true
}

func (h contentHandler) optionalUser(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	if h.content == nil {
		writeAppError(w, appError(http.StatusServiceUnavailable, "CONTENT_UNAVAILABLE", "content service is not configured"))
		return nil, false
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return nil, true
	}
	user, err := h.auth.AuthenticateAccessToken(r.Context(), token)
	if err != nil {
		writeAppError(w, authAppError(err))
		return nil, false
	}
	return &user, true
}

func intQuery(r *http.Request, key string) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func limitQuery(r *http.Request) int {
	limit := intQuery(r, "limit")
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}
