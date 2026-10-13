package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

// membershipKey is a private type, so only this package can read or write the value.
type membershipKey struct{}

// MembershipLoader finds out whether a user belongs to a workspace, and with
// which role. *services.WorkspaceService satisfies it.
type MembershipLoader interface {
	GetMembership(ctx context.Context, workspaceID, userID string) (*models.WorkspaceMembership, error)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// WorkspaceAccess guards every /workspaces/{workspaceID}/... route. It looks
// up the caller's membership in that workspace and stores it in the request.
// Non-members get a 404, so nobody can tell whether a workspace they are not
// in exists. It must run after Authenticate.
func WorkspaceAccess(loader MembershipLoader, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				unauthorized(w)
				return
			}

			workspaceID := chi.URLParam(r, "workspaceID")
			if !validate.UUID(workspaceID) {
				workspaceNotFound(w) // not even shaped like an id: do not bother the database
				return
			}

			membership, err := loader.GetMembership(r.Context(), workspaceID, userID)
			if err != nil {
				if errors.Is(err, services.ErrWorkspaceNotFound) {
					workspaceNotFound(w)
					return
				}
				logger.Error("loading workspace membership failed",
					"request_id", RequestIDFromContext(r.Context()),
					"error", err)
				response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
				return
			}

			ctx := context.WithValue(r.Context(), membershipKey{}, membership)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MembershipFromContext returns the caller's membership in the current
// workspace. The second result is false if WorkspaceAccess did not run.
func MembershipFromContext(ctx context.Context) (*models.WorkspaceMembership, bool) {
	m, ok := ctx.Value(membershipKey{}).(*models.WorkspaceMembership)
	return m, ok && m != nil
}

// RequirePermission only lets the request through if the caller's role in the
// current workspace includes the permission. It must run after WorkspaceAccess.
// If there is no membership at all, it refuses: it fails closed.
func RequirePermission(permission permissions.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m, ok := MembershipFromContext(r.Context())
			if !ok || !permissions.Can(m.Role, permission) {
				response.Error(w, http.StatusForbidden, response.CodeForbidden,
					"you do not have permission to do that")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func workspaceNotFound(w http.ResponseWriter) {
	response.Error(w, http.StatusNotFound, response.CodeNotFound, "workspace not found")
}
