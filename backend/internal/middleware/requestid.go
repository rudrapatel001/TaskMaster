package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// ctxKey is a private type, so no other package can accidentally use the same
// context key and overwrite our value.
type ctxKey struct{}

// RequestID gives every request a unique ID, stores it in the request context
// (for logs) and sets it on the response headers (for clients).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()

		w.Header().Set(response.HeaderRequestID, id)
		ctx := context.WithValue(r.Context(), ctxKey{}, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the current request's ID, or "" if there is none.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
