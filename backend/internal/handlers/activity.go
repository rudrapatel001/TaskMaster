package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type ActivityHandler struct {
	activity *services.ActivityService
	logger   *slog.Logger
}

func NewActivityHandler(activity *services.ActivityService, logger *slog.Logger) *ActivityHandler {
	return &ActivityHandler{activity: activity, logger: logger}
}

// Workspace handles GET /workspaces/{workspaceID}/activity.
//
// Query parameters (optional): limit (1 to 100, default 30) and before (the
// next_cursor of the previous page, to get older entries).
func (h *ActivityHandler) Workspace(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}
	limit, cursor, ok := h.paging(w, r)
	if !ok {
		return
	}

	page, err := h.activity.ListForWorkspace(r.Context(), actor.WorkspaceID, limit, cursor)
	h.respond(w, r, page, err)
}

// Task handles GET .../tasks/{number}/activity.
func (h *ActivityHandler) Task(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	projectID := chi.URLParam(r, "projectID")
	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if !validate.UUID(projectID) || err != nil || number < 1 {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
		return
	}
	limit, cursor, ok := h.paging(w, r)
	if !ok {
		return
	}

	page, err := h.activity.ListForTask(r.Context(), actor.WorkspaceID, projectID, number, limit, cursor)
	h.respond(w, r, page, err)
}

// paging reads and checks ?limit= and ?before=. It answers the request itself
// when something is wrong.
func (h *ActivityHandler) paging(w http.ResponseWriter, r *http.Request) (int, string, bool) {
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			response.ValidationError(w, map[string]string{"limit": "limit must be a number from 1 to 100"})
			return 0, "", false
		}
		limit = n
	}
	return limit, r.URL.Query().Get("before"), true
}

func (h *ActivityHandler) respond(w http.ResponseWriter, r *http.Request, page *repository.ActivityPage, err error) {
	switch {
	case err == nil:
		response.DataMeta(w, http.StatusOK, page.Events, map[string]any{"next_cursor": page.NextCursor})
	case errors.Is(err, services.ErrInvalidCursor):
		response.ValidationError(w, map[string]string{"before": "before is not a valid cursor"})
	case errors.Is(err, services.ErrTaskNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
	default:
		h.logger.Error("activity request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
