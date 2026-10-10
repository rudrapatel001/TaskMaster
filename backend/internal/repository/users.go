package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// ErrEmailTaken is returned when an account with the same email already exists.
var ErrEmailTaken = errors.New("email already taken")

// Postgres error code for "unique constraint violated".
const pgUniqueViolation = "23505"

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user and returns the stored row.
func (r *UserRepository) Create(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	const query = `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, name, password_hash, created_at, updated_at`

	var u models.User
	err := r.db.QueryRow(ctx, query, email, name, passwordHash).Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation &&
			pgErr.ConstraintName == "users_email_lower_idx" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &u, nil
}
