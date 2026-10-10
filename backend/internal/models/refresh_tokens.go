package models

import "time"

// RefreshToken is the stored record of a refresh token. It never contains the
// token itself, only facts about it.
type RefreshToken struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time // nil means "still active"
}
