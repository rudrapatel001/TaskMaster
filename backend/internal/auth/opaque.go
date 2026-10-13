package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// NewOpaqueToken creates a random, unguessable token (256 bits) for use in
// links, such as invitations. It returns the token itself, to give to the
// person, and its hash, the only thing we store. It is built the same way as
// refresh tokens.
func NewOpaqueToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashOpaqueToken(token), nil
}

// HashOpaqueToken returns the SHA-256 of a token, as hex.
func HashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
