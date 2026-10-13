package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type InvitationHandler struct {
	invitations *services.InvitationService
	logger      *slog.Logger
}

func NewInvitationHandler(invitations *services.InvitationService, logger *slog.Logger) *InvitationHandler {
	return &InvitationHandler{invitations: invitations, logger: logger}
}

type createInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// createdInvitation is the invitation plus its one-time token. Embedding
// models.Invitation makes the JSON flat.
type createdInvitation struct {
	models.Invitation
	Token string `json:"token"`
}

// Create handles POST /workspaces/{workspaceID}/invitations.
func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	var in createInvitationRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	role := models.Role(strings.TrimSpace(in.Role))

	v := validate.New()
	v.Check(validate.NotBlank(email), "email", "email is required")
	v.Check(validate.Email(email), "email", "enter a valid email address")
	v.Check(validate.OneOf(string(role), "admin", "member", "viewer"), "role", "role must be one of: admin, member, viewer")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	inv, token, err := h.invitations.Create(r.Context(), actor, email, role)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, createdInvitation{Invitation: *inv, Token: token})
}

// List handles GET /workspaces/{workspaceID}/invitations.
func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	list, err := h.invitations.ListPending(r.Context(), actor.WorkspaceID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, list)
}

// Revoke handles DELETE /workspaces/{workspaceID}/invitations/{invitationID}.
func (h *InvitationHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	invitationID := chi.URLParam(r, "invitationID")
	if !validate.UUID(invitationID) {
		h.notFound(w)
		return
	}

	if err := h.invitations.Revoke(r.Context(), actor.WorkspaceID, invitationID); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Preview handles GET /invitations/{token}. It needs no login: the person
// opening the link may not have an account yet.
func (h *InvitationHandler) Preview(w http.ResponseWriter, r *http.Request) {
	preview, err := h.invitations.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	response.Data(w, http.StatusOK, preview)
}

// Accept handles POST /invitations/{token}/accept. The caller must be logged in
// with the email address the invitation was sent to.
func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid access token")
		return
	}

	membership, err := h.invitations.Accept(r.Context(), chi.URLParam(r, "token"), userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, membership)
}

func (h *InvitationHandler) notFound(w http.ResponseWriter) {
	response.Error(w, http.StatusNotFound, response.CodeNotFound, "invitation not found or expired")
}

func (h *InvitationHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		h.notFound(w)
	case errors.Is(err, services.ErrWorkspaceNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "workspace not found")
	case errors.Is(err, services.ErrForbidden):
		response.Error(w, http.StatusForbidden, response.CodeForbidden,
			"your role does not allow inviting someone with that role")
	case errors.Is(err, services.ErrAlreadyMember):
		response.Error(w, http.StatusConflict, response.CodeConflict, "already a member of this workspace")
	case errors.Is(err, services.ErrInvitationEmailMismatch):
		response.Error(w, http.StatusForbidden, response.CodeForbidden,
			"this invitation was sent to a different email address: log in with that account")
	default:
		h.logger.Error("invitation request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
