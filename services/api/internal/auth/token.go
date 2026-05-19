package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type AccessClaims struct {
	UserID string
	Email  string
	Handle string
	Role   string
	Expiry time.Time
}

func newRefreshToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func signAccessToken(secret []byte, user User, now time.Time, ttl time.Duration) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]any{
		"sub":    user.ID,
		"email":  user.Email,
		"handle": user.Handle,
		"role":   user.Role,
		"exp":    now.Add(ttl).Unix(),
		"iat":    now.Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := hmacSHA256(secret, unsigned)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parseAccessToken(secret []byte, token string, now time.Time) (AccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AccessClaims{}, ErrUnauthorized
	}
	unsigned := parts[0] + "." + parts[1]
	want := hmacSHA256(secret, unsigned)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(got, want) {
		return AccessClaims{}, ErrUnauthorized
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AccessClaims{}, ErrUnauthorized
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return AccessClaims{}, ErrUnauthorized
	}

	exp, err := unixClaim(payload["exp"])
	if err != nil || !now.Before(exp) {
		return AccessClaims{}, ErrUnauthorized
	}

	return AccessClaims{
		UserID: stringClaim(payload["sub"]),
		Email:  stringClaim(payload["email"]),
		Handle: stringClaim(payload["handle"]),
		Role:   stringClaim(payload["role"]),
		Expiry: exp,
	}, nil
}

func hmacSHA256(secret []byte, value string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

func stringClaim(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func unixClaim(value any) (time.Time, error) {
	switch v := value.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	default:
		return time.Time{}, errors.New(fmt.Sprintf("unsupported unix claim %T", value))
	}
}
