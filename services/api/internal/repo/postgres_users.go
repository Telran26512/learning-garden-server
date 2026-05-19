package repo

import (
	"context"
	"errors"

	"github.com/Telran26512/learning-garden-server/services/api/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserStore struct {
	db *pgxpool.Pool
}

func NewPostgresUserStore(db *pgxpool.Pool) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

func (s *PostgresUserStore) CreateUser(ctx context.Context, input auth.CreateUserInput) (auth.User, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, handle, display_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, email::text, password_hash, handle::text, display_name, role, status, created_at
	`, input.ID, input.Email, input.PasswordHash, input.Handle, input.DisplayName)
	user, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return auth.User{}, auth.ErrEmailTaken
			case "users_handle_key":
				return auth.User{}, auth.ErrHandleTaken
			}
		}
		return auth.User{}, err
	}
	return user, nil
}

func (s *PostgresUserStore) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id::text, email::text, password_hash, handle::text, display_name, role, status, created_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrInvalidCredentials
	}
	return user, err
}

func (s *PostgresUserStore) FindByID(ctx context.Context, id string) (auth.User, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id::text, email::text, password_hash, handle::text, display_name, role, status, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUnauthorized
	}
	return user, err
}

type userRow interface {
	Scan(dest ...any) error
}

func scanUser(row userRow) (auth.User, error) {
	var user auth.User
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Handle,
		&user.DisplayName,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
	); err != nil {
		return auth.User{}, err
	}
	return user, nil
}
