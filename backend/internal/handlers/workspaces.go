package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type WorkspaceHandler struct {
	workspaces *services.WorkspaceService
	logger     *slog.Logger
}

func NewWorkspaceHandler(workspaces *services.WorkspaceService, logger *slog.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces, logger: logger}
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

// Create handles POST /workspaces. The caller becomes its owner.
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.unauthorized(w)
		return
	}

	var in createWorkspaceRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)

	v := validate.New()
	v.Check(validate.NotBlank(in.Name), "name", "name is required")
	v.Check(validate.MaxLen(in.Name, 60), "name", "name must be at most 60 characters")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	created, err := h.workspaces.Create(r.Context(), userID, in.Name)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			h.unauthorized(w) // valid token, but the account is gone
			return
		}
		h.logger.Error("create workspace failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		return
	}

	response.Data(w, http.StatusCreated, created)
}

// List handles GET /workspaces: the workspaces the caller belongs to.
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.unauthorized(w)
		return
	}

	list, err := h.workspaces.ListForUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("list workspaces failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		return
	}

	response.Data(w, http.StatusOK, list)
}

// Get handles GET /workspaces/{workspaceID}. WorkspaceAccess has already loaded
// the workspace and the caller's role, so there is nothing left to look up.
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership, ok := middleware.MembershipFromContext(r.Context())
	if !ok {
		h.forbidden(w)
		return
	}

	response.Data(w, http.StatusOK, membership)
}

// Rename handles PATCH /workspaces/{workspaceID}: change the workspace's name.
func (h *WorkspaceHandler) Rename(w http.ResponseWriter, r *http.Request) {
	membership, ok := middleware.MembershipFromContext(r.Context())
	if !ok {
		h.forbidden(w)
		return
	}

	var in createWorkspaceRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)

	v := validate.New()
	v.Check(validate.NotBlank(in.Name), "name", "name is required")
	v.Check(validate.MaxLen(in.Name, 60), "name", "name must be at most 60 characters")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	// Act on the workspace that was authorized (membership.ID), not on the raw URL.
	updated, err := h.workspaces.Rename(r.Context(), membership.ID, in.Name)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, models.WorkspaceMembership{Workspace: *updated, Role: membership.Role})
}

// Delete handles DELETE /workspaces/{workspaceID}. Only owners may do this.
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership, ok := middleware.MembershipFromContext(r.Context())
	if !ok {
		h.forbidden(w)
		return
	}

	if err := h.workspaces.Delete(r.Context(), membership.ID); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /workspaces/{workspaceID}/members.
func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	membership, ok := middleware.MembershipFromContext(r.Context())
	if !ok {
		h.forbidden(w)
		return
	}

	members, err := h.workspaces.ListMembers(r.Context(), membership.ID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, members)
}

// writeError turns a service error into a response.
func (h *WorkspaceHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, services.ErrWorkspaceNotFound) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "workspace not found")
		return
	}
	h.logger.Error("workspace request failed",
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"error", err)
	response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
}

func (h *WorkspaceHandler) forbidden(w http.ResponseWriter) {
	response.Error(w, http.StatusForbidden, response.CodeForbidden, "you do not have permission to do that")
}

func (h *WorkspaceHandler) unauthorized(w http.ResponseWriter) {
	response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
}
