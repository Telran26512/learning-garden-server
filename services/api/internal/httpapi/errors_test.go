package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteAppErrorUsesSingleEnvelopeShape(t *testing.T) {
	rec := httptest.NewRecorder()

	writeAppError(rec, AppError{
		Code:     "RATE_LIMITED",
		HTTPCode: http.StatusTooManyRequests,
		Message:  "too many requests",
		Err:      errors.New("internal limiter state"),
	})

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	var body struct {
		Data  any `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Meta map[string]any `json:"meta"`
	}
	decodeBody(t, rec, &body)
	if body.Error.Code != "RATE_LIMITED" {
		t.Fatalf("code = %q, want RATE_LIMITED", body.Error.Code)
	}
	if body.Error.Message != "too many requests" {
		t.Fatalf("message = %q, want too many requests", body.Error.Message)
	}
}

func TestWriteAppErrorDefaultsInternalErrors(t *testing.T) {
	rec := httptest.NewRecorder()

	writeAppError(rec, AppError{})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected error response body")
	}
}
