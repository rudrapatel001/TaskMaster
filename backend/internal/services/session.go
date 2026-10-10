package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rudrapatel001/taskmaster/backend/internal/auth"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// reuseGracePeriod is how long after a refresh token is used that presenting
// it again is treated as an innocent race (two tabs, a double click) rather
// than theft.
const reuseGracePeriod = 10 * time.Second

var (
	// ErrInvalidRefreshToken means the token is unknown, expired, or revoked.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrRefreshTokenReused means a long-since-used token was presented again,
	// which suggests it was stolen. All of that user's sessions are ended.
	ErrRefreshTokenReused = errors.New("refresh token reused")
)

// Refresh swaps a valid refresh token for a new session (rotation): the old
// token is revoked and a brand new access token and refresh token are issued.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	stored, err := s.refreshTokens.GetByHash(ctx, auth.HashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("looking up refresh token: %w", err)
	}

	if stored.RevokedAt != nil {
		// Used before. Right after use that is probably a harmless race; much
		// later it means someone is replaying an old token.
		if time.Since(*stored.RevokedAt) > reuseGracePeriod {
			if err := s.refreshTokens.RevokeAllForUser(ctx, stored.UserID); err != nil {
				return nil, fmt.Errorf("revoking sessions after token reuse: %w", err)
			}
			return nil, ErrRefreshTokenReused
		}
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	// Revoke the old token. If another request got there first, we lose the race.
	revoked, err := s.refreshTokens.Revoke(ctx, stored.ID)
	if err != nil {
		return nil, err
	}
	if !revoked {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	return s.newSession(ctx, user)
}

// Logout revokes a refresh token. It succeeds even if the token is unknown or
// already revoked, so logging out twice is harmless.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokens.RevokeByHash(ctx, auth.HashRefreshToken(refreshToken))
}
