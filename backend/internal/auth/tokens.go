package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "taskmaster"

// ErrInvalidToken is returned for any access token that is malformed, forged,
// expired, or otherwise not acceptable. Callers should not distinguish why.
var ErrInvalidToken = errors.New("invalid or expired token")

// TokenManager creates and checks short-lived access tokens (JWTs).
type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), accessTTL: accessTTL}
}

// NewAccessToken signs a JWT that identifies userID until it expires.
func (m *TokenManager) NewAccessToken(userID string) (token string, expiresAt time.Time, err error) {
	// JWT timestamps have one-second precision, so round now to match.
	now := time.Now().Truncate(time.Second)
	expiresAt = now.Add(m.accessTTL)

	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing access token: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseAccessToken checks the signature and expiry and returns the user ID inside.
func (m *TokenManager) ParseAccessToken(tokenString string) (userID string, err error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), // refuse any other algorithm, including "none"
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

// NewRefreshToken creates a random refresh token. It returns the token itself
// (to give to the client) and its hash (the only thing we store).
func NewRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32) // 256 bits of randomness
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken returns the SHA-256 of a refresh token, as hex.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
