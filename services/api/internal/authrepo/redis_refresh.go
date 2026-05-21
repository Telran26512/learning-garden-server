package authrepo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/redis/go-redis/v9"
)

type RedisRefreshStore struct {
	client *redis.Client
}

func NewRedisRefreshStore(client *redis.Client) *RedisRefreshStore {
	return &RedisRefreshStore{client: client}
}

func (s *RedisRefreshStore) Save(ctx context.Context, session auth.RefreshSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return auth.ErrRefreshRevoked
	}
	return s.client.Set(ctx, refreshKey(session.Token), payload, ttl).Err()
}

func (s *RedisRefreshStore) Find(ctx context.Context, token string) (auth.RefreshSession, error) {
	payload, err := s.client.Get(ctx, refreshKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return auth.RefreshSession{}, auth.ErrRefreshRevoked
	}
	if err != nil {
		return auth.RefreshSession{}, err
	}
	var session auth.RefreshSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return auth.RefreshSession{}, err
	}
	if session.Revoked {
		return auth.RefreshSession{}, auth.ErrRefreshRevoked
	}
	return session, nil
}

func (s *RedisRefreshStore) Revoke(ctx context.Context, token string) error {
	session, err := s.Find(ctx, token)
	if err != nil {
		return err
	}
	session.Revoked = true
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return auth.ErrRefreshRevoked
	}
	return s.client.Set(ctx, refreshKey(token), payload, ttl).Err()
}

func refreshKey(token string) string {
	return "refresh:" + token
}
