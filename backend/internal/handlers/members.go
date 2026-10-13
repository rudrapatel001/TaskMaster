package handlers

import (
	"errors"
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

type changeRoleRequest struct {
	Role string `json:"role"`
}

// ChangeMemberRole handles PATCH /workspaces/{workspaceID}/members/{userID}.
func (h *WorkspaceHandler) ChangeMemberRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		h.forbidden(w)
		return
	}

	targetID := chi.URLParam(r, "userID")
	if !validate.UUID(targetID) {
		h.memberNotFound(w)
		return
	}

	var in changeRoleRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	newRole := models.Role(strings.TrimSpace(in.Role))

	v := validate.New()
	v.Check(newRole.Valid(), "role", "role must be one of: viewer, member, admin, owner")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	updated, err := h.workspaces.ChangeRole(r.Context(), actor, targetID, newRole)
	if err != nil {
		h.writeMemberError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, updated)
}

// RemoveMember handles DELETE /workspaces/{workspaceID}/members/{userID}.
func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		h.forbidden(w)
		return
	}

	targetID := chi.URLParam(r, "userID")
	if !validate.UUID(targetID) {
		h.memberNotFound(w)
		return
	}
	if strings.EqualFold(targetID, actor.UserID) {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest,
			"to leave a workspace, use the leave action instead")
		return
	}

	if err := h.workspaces.RemoveMember(r.Context(), actor, targetID); err != nil {
		h.writeMemberError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Leave handles POST /workspaces/{workspaceID}/leave.
func (h *WorkspaceHandler) Leave(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		h.forbidden(w)
		return
	}

	if err := h.workspaces.Leave(r.Context(), actor); err != nil {
		h.writeMemberError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// actor describes who is making this request, inside this workspace. It relies
// on Authenticate and WorkspaceAccess having run.
func (h *WorkspaceHandler) actor(r *http.Request) (services.Actor, bool) {
	userID, hasUser := middleware.UserIDFromContext(r.Context())
	membership, hasMembership := middleware.MembershipFromContext(r.Context())
	if !hasUser || !hasMembership {
		return services.Actor{}, false
	}
	return services.Actor{WorkspaceID: membership.ID, UserID: userID, Role: membership.Role}, true
}

func (h *WorkspaceHandler) memberNotFound(w http.ResponseWriter) {
	response.Error(w, http.StatusNotFound, response.CodeNotFound, "member not found")
}

// writeMemberError turns a membership error into a response.
func (h *WorkspaceHandler) writeMemberError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrMemberNotFound):
		h.memberNotFound(w)
	case errors.Is(err, services.ErrForbidden):
		response.Error(w, http.StatusForbidden, response.CodeForbidden,
			"your role does not allow this change")
	case errors.Is(err, services.ErrLastOwner):
		response.Error(w, http.StatusConflict, response.CodeConflict,
			"a workspace must keep at least one owner: make someone else an owner first")
	case errors.Is(err, services.ErrMembershipChanged):
		response.Error(w, http.StatusConflict, response.CodeConflict,
			"the membership changed at the same moment, please try again")
	default:
		h.logger.Error("membership change failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
