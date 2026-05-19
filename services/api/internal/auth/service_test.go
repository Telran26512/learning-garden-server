package auth

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	service := NewService(NewMemoryUserStore(), NewMemoryRefreshStore(), TestConfig())
	ctx := context.Background()

	first, err := service.Register(ctx, RegisterInput{
		Email:       "zhe@example.dev",
		Handle:      "zhe-li",
		DisplayName: "李哲",
		Password:    "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if first.User.Email != "zhe@example.dev" {
		t.Fatalf("registered email = %q, want zhe@example.dev", first.User.Email)
	}

	_, err = service.Register(ctx, RegisterInput{
		Email:       "ZHE@example.dev",
		Handle:      "zhe-li-2",
		DisplayName: "李哲 2",
		Password:    "correct horse battery staple",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("second register err = %v, want ErrEmailTaken", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	service := NewService(NewMemoryUserStore(), NewMemoryRefreshStore(), TestConfig())
	ctx := context.Background()

	_, err := service.Register(ctx, RegisterInput{
		Email:       "aria@example.dev",
		Handle:      "aria-chen",
		DisplayName: "Aria Chen",
		Password:    "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = service.Login(ctx, LoginInput{
		Email:    "aria@example.dev",
		Password: "wrong password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Parallel()

	service := NewService(NewMemoryUserStore(), NewMemoryRefreshStore(), TestConfig())
	ctx := context.Background()

	session, err := service.Register(ctx, RegisterInput{
		Email:       "maxwell@example.dev",
		Handle:      "maxwell-tu",
		DisplayName: "Maxwell Tu",
		Password:    "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := service.Logout(ctx, session.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	_, err = service.Refresh(ctx, session.RefreshToken)
	if !errors.Is(err, ErrRefreshRevoked) {
		t.Fatalf("refresh err = %v, want ErrRefreshRevoked", err)
	}
}
