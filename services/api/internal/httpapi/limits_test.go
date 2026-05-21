package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
)

func TestListFeedGraphAndPortfolioEndpointsApplyDefaultAndMaxLimits(t *testing.T) {
	router := NewRouter(NewRouterConfig{
		Auth:    auth.NewService(auth.NewMemoryUserStore(), auth.NewMemoryRefreshStore(), auth.TestConfig()),
		Content: content.NewService(content.NewMemoryStore()),
	})
	owner := registerForTest(t, router, "limits@example.dev", "limits", "Limits User")

	for i := 0; i < 105; i++ {
		createContentForTest(t, router, owner.AccessToken, map[string]any{
			"kind":       "note",
			"title":      fmt.Sprintf("Public Note %03d", i),
			"slug":       fmt.Sprintf("public-note-%03d", i),
			"summary":    "Public note",
			"body":       "Public body",
			"visibility": "public",
			"status":     "published",
		})
	}

	publicDefault := doJSON(t, router, http.MethodGet, "/api/v1/content", nil, nil)
	assertItemsTotal(t, publicDefault, 20)

	publicMax := doJSON(t, router, http.MethodGet, "/api/v1/content?limit=999", nil, nil)
	assertItemsTotal(t, publicMax, 100)

	userContent := doJSON(t, router, http.MethodGet, "/api/v1/users/limits/content?limit=999", nil, nil)
	assertItemsTotal(t, userContent, 100)

	feed := doJSON(t, router, http.MethodGet, "/api/v1/community/feed?limit=999", nil, nil)
	assertItemsTotal(t, feed, 100)

	graph := doJSON(t, router, http.MethodGet, "/api/v1/graph?handle=limits&limit=999", nil, nil)
	var graphBody struct {
		Data content.Graph `json:"data"`
	}
	decodeBody(t, graph, &graphBody)
	if len(graphBody.Data.Nodes) != 100 {
		t.Fatalf("graph nodes = %d, want 100", len(graphBody.Data.Nodes))
	}

	portfolio := doJSON(t, router, http.MethodGet, "/api/v1/portfolio/limits?limit=999", nil, nil)
	var portfolioBody struct {
		Data content.Portfolio `json:"data"`
	}
	decodeBody(t, portfolio, &portfolioBody)
	if got := portfolioItemCount(portfolioBody.Data); got != 100 {
		t.Fatalf("portfolio item count = %d, want 100", got)
	}
	if len(portfolioBody.Data.Graph.Nodes) != 100 {
		t.Fatalf("portfolio graph nodes = %d, want 100", len(portfolioBody.Data.Graph.Nodes))
	}
}

func assertItemsTotal(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeBody(t, rec, &body)
	if body.Data.Total != want {
		t.Fatalf("total = %d, want %d, body = %s", body.Data.Total, want, rec.Body.String())
	}
}

func portfolioItemCount(portfolio content.Portfolio) int {
	total := 0
	for _, items := range portfolio.Items {
		total += len(items)
	}
	return total
}
