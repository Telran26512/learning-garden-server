package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
)

func TestAuthEndpointsRegisterLoginMeRefreshAndLogout(t *testing.T) {
	t.Parallel()

	router := NewRouter(NewRouterConfig{
		Auth: auth.NewService(auth.NewMemoryUserStore(), auth.NewMemoryRefreshStore(), auth.TestConfig()),
	})

	register := doJSON(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":       "zhe@example.dev",
		"handle":      "zhe-li",
		"displayName": "李哲",
		"password":    "correct horse battery staple",
	}, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", register.Code, register.Body.String())
	}
	refreshCookie := findCookie(register, "synapse_refresh")
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatalf("register did not set synapse_refresh cookie")
	}

	var registerBody struct {
		Data struct {
			AccessToken string `json:"accessToken"`
			User        struct {
				Email  string `json:"email"`
				Handle string `json:"handle"`
			} `json:"user"`
		} `json:"data"`
	}
	decodeBody(t, register, &registerBody)
	if registerBody.Data.AccessToken == "" {
		t.Fatalf("register did not return access token")
	}
	if registerBody.Data.User.Email != "zhe@example.dev" || registerBody.Data.User.Handle != "zhe-li" {
		t.Fatalf("registered user = %+v", registerBody.Data.User)
	}

	me := doJSON(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + registerBody.Data.AccessToken,
	})
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}

	login := doJSON(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "zhe@example.dev",
		"password": "correct horse battery staple",
	}, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}
	loginCookie := findCookie(login, "synapse_refresh")
	if loginCookie == nil || loginCookie.Value == "" {
		t.Fatalf("login did not set synapse_refresh cookie")
	}

	refresh := doJSON(t, router, http.MethodPost, "/auth/refresh", nil, map[string]string{
		"Cookie": loginCookie.String(),
	})
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", refresh.Code, refresh.Body.String())
	}

	logout := doJSON(t, router, http.MethodPost, "/auth/logout", nil, map[string]string{
		"Cookie": loginCookie.String(),
	})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logout.Code, logout.Body.String())
	}

	revoked := doJSON(t, router, http.MethodPost, "/auth/refresh", nil, map[string]string{
		"Cookie": loginCookie.String(),
	})
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want 401, body = %s", revoked.Code, revoked.Body.String())
	}
}

func doJSON(t *testing.T, handler http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
