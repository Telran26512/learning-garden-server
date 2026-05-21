package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
)

func TestP2ContentPortfolioCommunityAndGraphEndpoints(t *testing.T) {
	t.Parallel()

	router := NewRouter(NewRouterConfig{
		Auth:    auth.NewService(auth.NewMemoryUserStore(), auth.NewMemoryRefreshStore(), auth.TestConfig()),
		Content: content.NewService(content.NewMemoryStore()),
	})

	owner := registerForTest(t, router, "xiaobin@example.dev", "xiaobin-cao", "Xiaobin Cao")
	other := registerForTest(t, router, "reader@example.dev", "reader", "Reader")

	privateNote := createContentForTest(t, router, owner.AccessToken, map[string]any{
		"kind":       "note",
		"title":      "Private scratchpad",
		"slug":       "private-scratchpad",
		"summary":    "Draft notes that should not show up publicly.",
		"body":       "private draft",
		"visibility": "private",
		"status":     "draft",
		"metadata": map[string]any{
			"tags": []string{"draft"},
		},
	})
	track := createContentForTest(t, router, owner.AccessToken, map[string]any{
		"kind":       "track",
		"title":      "Transformer 精读",
		"slug":       "transformer-reading",
		"summary":    "从 Q/K/V 到 FlashAttention 的学习路线。",
		"body":       "track body",
		"visibility": "public",
		"status":     "shipped",
		"metadata": map[string]any{
			"tags":     []string{"Transformer", "attention"},
			"progress": 100,
		},
	})
	note := createContentForTest(t, router, owner.AccessToken, map[string]any{
		"kind":       "note",
		"title":      "Scaled Dot-Product 的几何直觉",
		"slug":       "scaled-dot-product-geometry",
		"summary":    "解释 QK^T / sqrt(d_k) 的方差与几何意义。",
		"body":       "把 Q 和 K 看作查询向量与键向量。",
		"visibility": "public",
		"status":     "published",
		"metadata": map[string]any{
			"tags":   []string{"Transformer", "math"},
			"blocks": 3,
			"links":  7,
		},
	})
	paper := createContentForTest(t, router, owner.AccessToken, map[string]any{
		"kind":       "paper",
		"title":      "Attention Is All You Need",
		"slug":       "attention-is-all-you-need",
		"summary":    "Transformer 原论文精读。",
		"body":       "paper notes",
		"visibility": "public",
		"status":     "done",
		"metadata": map[string]any{
			"authors": []string{"Vaswani et al."},
			"arxiv":   "1706.03762",
		},
	})

	link := doJSON(t, router, http.MethodPost, "/api/v1/content/"+note.ID+"/relations", map[string]string{
		"targetId": track.ID,
		"type":     "belongs_to",
	}, map[string]string{
		"Authorization": "Bearer " + owner.AccessToken,
	})
	if link.Code != http.StatusCreated {
		t.Fatalf("link status = %d, body = %s", link.Code, link.Body.String())
	}
	linkPaper := doJSON(t, router, http.MethodPost, "/api/v1/content/"+note.ID+"/relations", map[string]string{
		"targetId": paper.ID,
		"type":     "cites",
	}, map[string]string{
		"Authorization": "Bearer " + owner.AccessToken,
	})
	if linkPaper.Code != http.StatusCreated {
		t.Fatalf("paper link status = %d, body = %s", linkPaper.Code, linkPaper.Body.String())
	}

	publicNotes := doJSON(t, router, http.MethodGet, "/api/v1/content?kind=note", nil, nil)
	if publicNotes.Code != http.StatusOK {
		t.Fatalf("public notes status = %d, body = %s", publicNotes.Code, publicNotes.Body.String())
	}
	body := publicNotes.Body.String()
	if !strings.Contains(body, note.Title) {
		t.Fatalf("public notes did not include published note: %s", body)
	}
	if strings.Contains(body, privateNote.Title) {
		t.Fatalf("public notes leaked private note: %s", body)
	}

	patchByOther := doJSON(t, router, http.MethodPatch, "/api/v1/content/"+note.ID, map[string]string{
		"title": "hijacked",
	}, map[string]string{
		"Authorization": "Bearer " + other.AccessToken,
	})
	if patchByOther.Code != http.StatusForbidden {
		t.Fatalf("patch by non-owner status = %d, want 403, body = %s", patchByOther.Code, patchByOther.Body.String())
	}

	profile := doJSON(t, router, http.MethodGet, "/api/v1/users/xiaobin-cao", nil, nil)
	if profile.Code != http.StatusOK {
		t.Fatalf("profile status = %d, body = %s", profile.Code, profile.Body.String())
	}
	if !strings.Contains(profile.Body.String(), `"displayName":"Xiaobin Cao"`) {
		t.Fatalf("profile missing display name: %s", profile.Body.String())
	}

	portfolio := doJSON(t, router, http.MethodGet, "/api/v1/portfolio/xiaobin-cao", nil, nil)
	if portfolio.Code != http.StatusOK {
		t.Fatalf("portfolio status = %d, body = %s", portfolio.Code, portfolio.Body.String())
	}
	portfolioBody := portfolio.Body.String()
	for _, required := range []string{track.Title, note.Title, paper.Title, `"nodes"`, `"edges"`} {
		if !strings.Contains(portfolioBody, required) {
			t.Fatalf("portfolio missing %q: %s", required, portfolioBody)
		}
	}
	if strings.Contains(portfolioBody, privateNote.Title) {
		t.Fatalf("portfolio leaked private content: %s", portfolioBody)
	}

	backlinks := doJSON(t, router, http.MethodGet, "/api/v1/content/"+track.ID+"/backlinks", nil, nil)
	if backlinks.Code != http.StatusOK {
		t.Fatalf("backlinks status = %d, body = %s", backlinks.Code, backlinks.Body.String())
	}
	if !strings.Contains(backlinks.Body.String(), note.Title) {
		t.Fatalf("backlinks missing source note: %s", backlinks.Body.String())
	}

	feed := doJSON(t, router, http.MethodGet, "/api/v1/community/feed", nil, nil)
	if feed.Code != http.StatusOK {
		t.Fatalf("feed status = %d, body = %s", feed.Code, feed.Body.String())
	}
	if !strings.Contains(feed.Body.String(), note.Title) || strings.Contains(feed.Body.String(), privateNote.Title) {
		t.Fatalf("feed visibility mismatch: %s", feed.Body.String())
	}

	remove := doJSON(t, router, http.MethodDelete, "/api/v1/content/"+privateNote.ID, nil, map[string]string{
		"Authorization": "Bearer " + owner.AccessToken,
	})
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

type registeredUser struct {
	AccessToken string
}

func registerForTest(t *testing.T, router http.Handler, email, handle, displayName string) registeredUser {
	t.Helper()
	rec := doJSON(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":       email,
		"handle":      handle,
		"displayName": displayName,
		"password":    "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d, body = %s", handle, rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	decodeBody(t, rec, &body)
	return registeredUser{AccessToken: body.Data.AccessToken}
}

func createContentForTest(t *testing.T, router http.Handler, token string, body map[string]any) content.Item {
	t.Helper()
	rec := doJSON(t, router, http.MethodPost, "/api/v1/content", body, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create content status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data content.Item `json:"data"`
	}
	decodeBody(t, rec, &out)
	return out.Data
}
