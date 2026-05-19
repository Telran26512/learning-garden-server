package auth

import "errors"

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrHandleTaken        = errors.New("handle already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrRefreshRevoked     = errors.New("refresh token revoked")
	ErrUnauthorized       = errors.New("unauthorized")
)
