package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
)

type AppError struct {
	Code     string
	HTTPCode int
	Message  string
	Err      error
}

func (e AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e AppError) Unwrap() error {
	return e.Err
}

func writeAppError(w http.ResponseWriter, appErr AppError) {
	if appErr.HTTPCode == 0 {
		appErr.HTTPCode = http.StatusInternalServerError
	}
	if appErr.Code == "" {
		appErr.Code = "INTERNAL"
	}
	if appErr.Message == "" {
		appErr.Message = "internal server error"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": nil,
		"error": map[string]any{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
		"meta": map[string]any{},
	})
}

func appError(httpCode int, code string, message string) AppError {
	return AppError{HTTPCode: httpCode, Code: code, Message: message}
}

func authAppError(err error) AppError {
	switch {
	case errors.Is(err, auth.ErrEmailTaken), errors.Is(err, auth.ErrHandleTaken):
		return AppError{HTTPCode: http.StatusConflict, Code: "CONFLICT", Message: err.Error(), Err: err}
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrRefreshRevoked), errors.Is(err, auth.ErrUnauthorized):
		return AppError{HTTPCode: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: err.Error(), Err: err}
	default:
		return AppError{HTTPCode: http.StatusInternalServerError, Code: "INTERNAL", Message: "internal server error", Err: err}
	}
}

func contentAppError(err error) AppError {
	switch {
	case errors.Is(err, content.ErrInvalidInput):
		return AppError{HTTPCode: http.StatusBadRequest, Code: "BAD_REQUEST", Message: err.Error(), Err: err}
	case errors.Is(err, content.ErrForbidden):
		return AppError{HTTPCode: http.StatusForbidden, Code: "FORBIDDEN", Message: err.Error(), Err: err}
	case errors.Is(err, content.ErrNotFound):
		return AppError{HTTPCode: http.StatusNotFound, Code: "NOT_FOUND", Message: err.Error(), Err: err}
	case errors.Is(err, content.ErrConflict):
		return AppError{HTTPCode: http.StatusConflict, Code: "CONFLICT", Message: err.Error(), Err: err}
	case errors.Is(err, auth.ErrUnauthorized):
		return AppError{HTTPCode: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: err.Error(), Err: err}
	default:
		return AppError{HTTPCode: http.StatusInternalServerError, Code: "INTERNAL", Message: "internal server error", Err: err}
	}
}
