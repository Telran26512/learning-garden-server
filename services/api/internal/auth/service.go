package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

type Config struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecret       string
}

func DefaultConfig() Config {
	return Config{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		JWTSecret:       "dev-only-change-me",
	}
}

func TestConfig() Config {
	cfg := DefaultConfig()
	cfg.JWTSecret = "test-secret"
	return cfg
}

type RegisterInput struct {
	InviteCode  string `json:"inviteCode"`
	Email       string `json:"email"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Handle       string    `json:"handle"`
	DisplayName  string    `json:"displayName"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"-"`
	User         User   `json:"user"`
}

type CreateUserInput struct {
	ID           string
	Email        string
	Handle       string
	DisplayName  string
	PasswordHash string
}

type UserStore interface {
	CreateUser(context.Context, CreateUserInput) (User, error)
	FindByEmail(context.Context, string) (User, error)
	FindByHandle(context.Context, string) (User, error)
	FindByID(context.Context, string) (User, error)
}

type RefreshSession struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
}

type RefreshStore interface {
	Save(context.Context, RefreshSession) error
	Find(context.Context, string) (RefreshSession, error)
	Revoke(context.Context, string) error
}

type Service struct {
	users    UserStore
	refresh  RefreshStore
	config   Config
	now      func() time.Time
	tokenTTL time.Duration
}

func NewService(users UserStore, refresh RefreshStore, config Config) *Service {
	if config.AccessTokenTTL == 0 {
		config.AccessTokenTTL = DefaultConfig().AccessTokenTTL
	}
	if config.RefreshTokenTTL == 0 {
		config.RefreshTokenTTL = DefaultConfig().RefreshTokenTTL
	}
	if config.JWTSecret == "" {
		config.JWTSecret = DefaultConfig().JWTSecret
	}
	return &Service{
		users:   users,
		refresh: refresh,
		config:  config,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (Session, error) {
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return Session{}, err
	}
	user, err := s.users.CreateUser(ctx, CreateUserInput{
		ID:           randomID(),
		Email:        input.Email,
		Handle:       input.Handle,
		DisplayName:  input.DisplayName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return Session{}, err
	}
	return s.newSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (Session, error) {
	user, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	if user.Status != "active" || !verifyPassword(input.Password, user.PasswordHash) {
		return Session{}, ErrInvalidCredentials
	}
	return s.newSession(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	stored, err := s.refresh.Find(ctx, refreshToken)
	if err != nil {
		return Session{}, ErrRefreshRevoked
	}
	if !s.now().Before(stored.ExpiresAt) {
		return Session{}, ErrRefreshRevoked
	}
	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	accessToken, err := signAccessToken([]byte(s.config.JWTSecret), user, s.now(), s.config.AccessTokenTTL)
	if err != nil {
		return Session{}, err
	}
	return Session{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.refresh.Revoke(ctx, refreshToken)
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (User, error) {
	claims, err := parseAccessToken([]byte(s.config.JWTSecret), accessToken, s.now())
	if err != nil {
		return User{}, err
	}
	return s.users.FindByID(ctx, claims.UserID)
}

func (s *Service) FindUserByHandle(ctx context.Context, handle string) (User, error) {
	return s.users.FindByHandle(ctx, handle)
}

func (s *Service) newSession(ctx context.Context, user User) (Session, error) {
	accessToken, err := signAccessToken([]byte(s.config.JWTSecret), user, s.now(), s.config.AccessTokenTTL)
	if err != nil {
		return Session{}, err
	}
	refreshToken, err := newRefreshToken()
	if err != nil {
		return Session{}, err
	}
	if err := s.refresh.Save(ctx, RefreshSession{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: s.now().Add(s.config.RefreshTokenTTL),
	}); err != nil {
		return Session{}, err
	}
	return Session{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

func randomID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(raw)
}
