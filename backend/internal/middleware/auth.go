package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/auth"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// userIDKey is a private type, so only this package can read or write the value.
type userIDKey struct{}

// Authenticate protects routes: a request must carry a valid access token in
// an "Authorization: Bearer <token>" header. The user's ID is then available
// to handlers through UserIDFromContext.
func Authenticate(tokens *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				unauthorized(w)
				return
			}

			userID, err := tokens.ParseAccessToken(token)
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the ID of the logged-in user. The second result is
// false if the request did not go through Authenticate.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok && id != ""
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
}
