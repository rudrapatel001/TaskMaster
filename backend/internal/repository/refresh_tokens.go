package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
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

// GetByHash finds the record for a token hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	const query = `
		SELECT id, user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var t models.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &t.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying refresh token: %w", err)
	}
	return &t, nil
}

// Revoke marks one token as revoked. It reports whether this call was the one
// that revoked it (false means it was already revoked, for example by a
// simultaneous request).
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) (bool, error) {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("revoking refresh token: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// RevokeByHash revokes the token with this hash, if it exists and is active.
func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := r.db.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoking refresh token by hash: %w", err)
	}
	return nil
}

// RevokeAllForUser ends every active session of one user.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`

	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("revoking all refresh tokens: %w", err)
	}
	return nil
}
