package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

var statusCategories = []string{"todo", "in_progress", "done"}

type StatusHandler struct {
	statuses *services.StatusService
	logger   *slog.Logger
}

func NewStatusHandler(statuses *services.StatusService, logger *slog.Logger) *StatusHandler {
	return &StatusHandler{statuses: statuses, logger: logger}
}

type createStatusRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

// updateStatusRequest uses pointers: a field that is not sent stays as it is.
type updateStatusRequest struct {
	Name     *string `json:"name"`
	Category *string `json:"category"`
}

type moveStatusRequest struct {
	Before *string `json:"before"` // put the column directly before this one; null means last
}

// Create handles POST .../projects/{projectID}/statuses.
func (h *StatusHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.project(w, r)
	if !ok {
		return
	}

	var in createStatusRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)

	v := validate.New()
	checkStatusName(v, in.Name)
	v.Check(validate.OneOf(in.Category, statusCategories...), "category", "category must be one of: todo, in_progress, done")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	created, err := h.statuses.Create(r.Context(), actor.WorkspaceID, projectID, in.Name, in.Category)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, created)
}

// Update handles PATCH .../statuses/{statusID}: rename a column and/or change its category.
func (h *StatusHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, projectID, statusID, ok := h.status(w, r)
	if !ok {
		return
	}

	var in updateStatusRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}

	v := validate.New()
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		in.Name = &trimmed
		checkStatusName(v, trimmed)
	}
	if in.Category != nil {
		v.Check(validate.OneOf(*in.Category, statusCategories...), "category", "category must be one of: todo, in_progress, done")
	}
	if in.Name == nil && in.Category == nil {
		v.Check(false, "body", "send at least one field to change")
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	updated, err := h.statuses.Update(r.Context(), actor.WorkspaceID, projectID, statusID, in.Name, in.Category)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, updated)
}

// Move handles POST .../statuses/{statusID}/move and returns all the columns in their new order.
func (h *StatusHandler) Move(w http.ResponseWriter, r *http.Request) {
	actor, projectID, statusID, ok := h.status(w, r)
	if !ok {
		return
	}

	var in moveStatusRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	if in.Before != nil && !validate.UUID(*in.Before) {
		response.ValidationError(w, map[string]string{"before": "before must be a valid id, or null for last"})
		return
	}

	statuses, err := h.statuses.Move(r.Context(), actor.WorkspaceID, projectID, statusID, in.Before)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, statuses)
}

// Delete handles DELETE .../statuses/{statusID}?move_to={otherStatusID}.
// Tasks in the column move to move_to; it is required if the column has tasks.
func (h *StatusHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, projectID, statusID, ok := h.status(w, r)
	if !ok {
		return
	}

	var moveTo *string
	if raw := r.URL.Query().Get("move_to"); raw != "" {
		if !validate.UUID(raw) {
			response.ValidationError(w, map[string]string{"move_to": "move_to must be a valid id"})
			return
		}
		moveTo = &raw
	}

	if err := h.statuses.Delete(r.Context(), actor.WorkspaceID, projectID, statusID, moveTo); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func checkStatusName(v *validate.Validator, name string) {
	v.Check(validate.NotBlank(name), "name", "name is required")
	v.Check(validate.MaxLen(name, 50), "name", "name must be at most 50 characters")
}

// project finds who is asking and which project they mean.
func (h *StatusHandler) project(w http.ResponseWriter, r *http.Request) (services.Actor, string, bool) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return services.Actor{}, "", false
	}

	projectID := chi.URLParam(r, "projectID")
	if !validate.UUID(projectID) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "project not found")
		return services.Actor{}, "", false
	}
	return actor, projectID, true
}

// status also reads the column's id. A malformed id is answered like an unknown one.
func (h *StatusHandler) status(w http.ResponseWriter, r *http.Request) (services.Actor, string, string, bool) {
	actor, projectID, ok := h.project(w, r)
	if !ok {
		return services.Actor{}, "", "", false
	}

	statusID := chi.URLParam(r, "statusID")
	if !validate.UUID(statusID) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "column not found")
		return services.Actor{}, "", "", false
	}
	return actor, projectID, statusID, true
}

func (h *StatusHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "project not found")
	case errors.Is(err, services.ErrStatusNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "column not found")
	case errors.Is(err, services.ErrStatusNameTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict, "a column with this name already exists in the project")
	case errors.Is(err, services.ErrTooManyStatuses):
		response.Error(w, http.StatusConflict, response.CodeConflict,
			fmt.Sprintf("a project can have at most %d columns", repository.MaxStatuses))
	case errors.Is(err, services.ErrLastStatus):
		response.Error(w, http.StatusConflict, response.CodeConflict, "a project must keep at least one column")
	case errors.Is(err, services.ErrColumnNotEmpty):
		response.Error(w, http.StatusConflict, response.CodeConflict,
			"this column still has tasks: choose another column to move them to")
	case errors.Is(err, services.ErrInvalidTarget):
		response.ValidationError(w, map[string]string{"move_to": "must be another column of this project"})
	case errors.Is(err, services.ErrInvalidStatusNeighbour):
		response.ValidationError(w, map[string]string{"before": "must be another column of this project"})
	default:
		h.logger.Error("column request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
