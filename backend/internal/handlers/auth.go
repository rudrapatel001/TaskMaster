package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type AuthHandler struct {
	auth   *services.AuthService
	logger *slog.Logger
}

func NewAuthHandler(auth *services.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
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
