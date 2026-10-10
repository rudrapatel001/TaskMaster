package handlers

import (
	"errors"
	"net/http"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
)

// Refresh handles POST /auth/refresh: it trades the refresh cookie for a new
// access token and a new refresh cookie.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.rejectRefresh(w)
		return
	}

	session, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRefreshTokenReused):
			h.logger.Warn("refresh token reuse detected: all sessions for the user were revoked",
				"request_id", middleware.RequestIDFromContext(r.Context()))
			h.rejectRefresh(w)
		case errors.Is(err, services.ErrInvalidRefreshToken):
			h.rejectRefresh(w)
		default:
			h.logger.Error("refresh failed",
				"request_id", middleware.RequestIDFromContext(r.Context()),
				"error", err)
			response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		}
		return
	}

	h.setRefreshCookie(w, session.RefreshToken, session.RefreshTokenExpiresAt)
	w.Header().Set("Cache-Control", "no-store")

	response.Data(w, http.StatusOK, sessionResponse{
		User:        session.User,
		AccessToken: session.AccessToken,
		ExpiresAt:   session.AccessTokenExpiresAt,
	})
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			h.logger.Error("logout failed",
				"request_id", middleware.RequestIDFromContext(r.Context()),
				"error", err)
			response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
			return
		}
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// rejectRefresh answers a failed refresh with a 401 and removes the useless cookie.
// Every failure looks the same from outside, whatever the reason.
func (h *AuthHandler) rejectRefresh(w http.ResponseWriter) {
	h.clearRefreshCookie(w)
	response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "session expired, please log in again")
}

// clearRefreshCookie tells the browser to delete the refresh cookie.
func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath, // must match the path it was set with
		MaxAge:   -1,                // delete now
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}
