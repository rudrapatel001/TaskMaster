package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

const (
	refreshCookieName = "refresh_token"
	// The browser only sends this cookie to URLs starting with /auth.
	refreshCookiePath = "/auth"
)

type AuthHandler struct {
	auth          *services.AuthService
	logger        *slog.Logger
	secureCookies bool
}

func NewAuthHandler(auth *services.AuthService, logger *slog.Logger, secureCookies bool) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger, secureCookies: secureCookies}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in registerRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}

	// Normalize first, so validation and storage see the same values.
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)

	v := validate.New()
	v.Check(validate.NotBlank(in.Email), "email", "email is required")
	v.Check(validate.Email(in.Email), "email", "enter a valid email address")
	v.Check(validate.NotBlank(in.Name), "name", "name is required")
	v.Check(validate.MaxLen(in.Name, 100), "name", "name must be at most 100 characters")
	v.Check(in.Password != "", "password", "password is required")
	v.Check(validate.MinLen(in.Password, 8), "password", "password must be at least 8 characters")
	v.Check(validate.MaxBytes(in.Password, 72), "password", "password is too long (72 bytes maximum)")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	user, err := h.auth.Register(r.Context(), services.RegisterInput{
		Email:    in.Email,
		Name:     in.Name,
		Password: in.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			response.Error(w, http.StatusConflict, response.CodeConflict,
				"an account with this email already exists")
			return
		}
		h.logger.Error("register failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		return
	}

	response.Data(w, http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	User        *models.User `json:"user"`
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	// Only check that something was sent. Never apply password rules here:
	// that would tell an attacker what a valid password looks like.
	v := validate.New()
	v.Check(validate.NotBlank(in.Email), "email", "email is required")
	v.Check(in.Password != "", "password", "password is required")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	session, err := h.auth.Login(r.Context(), services.LoginInput{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized,
				"invalid email or password")
			return
		}
		h.logger.Error("login failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		return
	}

	h.setRefreshCookie(w, session.RefreshToken, session.RefreshTokenExpiresAt)

	// Responses that contain tokens must never be stored by caches.
	w.Header().Set("Cache-Control", "no-store")

	response.Data(w, http.StatusOK, sessionResponse{
		User:        session.User,
		AccessToken: session.AccessToken,
		ExpiresAt:   session.AccessTokenExpiresAt,
	})
}

// setRefreshCookie puts the refresh token in a cookie JavaScript cannot read.
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(time.Until(expiresAt).Round(time.Second).Seconds()),
		HttpOnly: true,                    // JavaScript cannot read it, so an XSS bug cannot steal it
		Secure:   h.secureCookies,         // HTTPS only (in production)
		SameSite: http.SameSiteStrictMode, // not sent on requests started from other sites
	})
}
