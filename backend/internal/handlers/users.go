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

type UserHandler struct {
	users  *services.UserService
	logger *slog.Logger
}

func NewUserHandler(users *services.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{users: users, logger: logger}
}

// Me handles GET /me: the profile of the logged-in user.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
		return
	}

	user, err := h.users.Get(r.Context(), userID)
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

type updateMeRequest struct {
	Name string `json:"name"`
}

// UpdateMe handles PATCH /me: change the logged-in user's name.
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
		return
	}

	var in updateMeRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)

	v := validate.New()
	v.Check(validate.NotBlank(in.Name), "name", "name is required")
	v.Check(validate.MaxLen(in.Name, 100), "name", "name must be at most 100 characters")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	user, err := h.users.UpdateName(r.Context(), userID, in.Name)
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

func (h *UserHandler) writeUserError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, services.ErrUserNotFound) {
		// The token was valid but the account is gone, so the token is useless.
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
		return
	}
	h.logger.Error("user request failed",
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"error", err)
	response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
}
