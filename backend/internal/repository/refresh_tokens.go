package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create stores the hash of a new refresh token.
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("inserting refresh token: %w", err)
	}
	return nil
}
