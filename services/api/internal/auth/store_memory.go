package auth

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryUserStore struct {
	mu       sync.Mutex
	users    map[string]User
	byEmail  map[string]string
	byHandle map[string]string
	nextID   int
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users:    map[string]User{},
		byEmail:  map[string]string{},
		byHandle: map[string]string{},
	}
}

func (s *MemoryUserStore) CreateUser(_ context.Context, input CreateUserInput) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email := normalizeEmail(input.Email)
	handle := normalizeHandle(input.Handle)
	if _, ok := s.byEmail[email]; ok {
		return User{}, ErrEmailTaken
	}
	if _, ok := s.byHandle[handle]; ok {
		return User{}, ErrHandleTaken
	}

	s.nextID++
	user := User{
		ID:           strings.TrimSpace(input.ID),
		Email:        email,
		Handle:       handle,
		DisplayName:  strings.TrimSpace(input.DisplayName),
		PasswordHash: input.PasswordHash,
		Role:         "user",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}
	if user.ID == "" {
		user.ID = randomID()
	}
	s.users[user.ID] = user
	s.byEmail[email] = user.ID
	s.byHandle[handle] = user.ID
	return user, nil
}

func (s *MemoryUserStore) FindByEmail(_ context.Context, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.byEmail[normalizeEmail(email)]
	if !ok {
		return User{}, ErrInvalidCredentials
	}
	return s.users[id], nil
}

func (s *MemoryUserStore) FindByHandle(_ context.Context, handle string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.byHandle[normalizeHandle(handle)]
	if !ok {
		return User{}, ErrUnauthorized
	}
	return s.users[id], nil
}

func (s *MemoryUserStore) FindByID(_ context.Context, id string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return User{}, ErrUnauthorized
	}
	return user, nil
}

type MemoryRefreshStore struct {
	mu       sync.Mutex
	sessions map[string]RefreshSession
}

func NewMemoryRefreshStore() *MemoryRefreshStore {
	return &MemoryRefreshStore{sessions: map[string]RefreshSession{}}
}

func (s *MemoryRefreshStore) Save(_ context.Context, session RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.Token] = session
	return nil
}

func (s *MemoryRefreshStore) Find(_ context.Context, token string) (RefreshSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok || session.Revoked {
		return RefreshSession{}, ErrRefreshRevoked
	}
	return session, nil
}

func (s *MemoryRefreshStore) Revoke(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok {
		return ErrRefreshRevoked
	}
	session.Revoked = true
	s.sessions[token] = session
	return nil
}
