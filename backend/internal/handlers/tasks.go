package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

var priorities = []string{"low", "medium", "high", "urgent"}

type TaskHandler struct {
	tasks  *services.TaskService
	logger *slog.Logger
}

func NewTaskHandler(tasks *services.TaskService, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{tasks: tasks, logger: logger}
}

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StatusID    string  `json:"status_id"` // optional: defaults to the first column
	Priority    string  `json:"priority"`  // optional: defaults to medium
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

// Create handles POST /workspaces/{workspaceID}/projects/{projectID}/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.project(w, r)
	if !ok {
		return
	}

	var in createTaskRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Priority == "" {
		in.Priority = "medium"
	}

	v := validate.New()
	checkTitle(v, in.Title)
	checkDescription(v, in.Description)
	v.Check(validate.OneOf(in.Priority, priorities...), "priority", "priority must be one of: low, medium, high, urgent")
	v.Check(in.StatusID == "" || validate.UUID(in.StatusID), "status_id", "status_id must be a valid id")
	if in.AssigneeID != nil {
		v.Check(validate.UUID(*in.AssigneeID), "assignee_id", "assignee_id must be a valid id")
	}
	if in.DueDate != nil {
		v.Check(validDate(*in.DueDate), "due_date", "due_date must look like 2026-10-20")
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	created, err := h.tasks.Create(r.Context(), repository.NewTask{
		WorkspaceID: actor.WorkspaceID,
		ProjectID:   projectID,
		ReporterID:  actor.UserID,
		Title:       in.Title,
		Description: in.Description,
		StatusID:    in.StatusID,
		Priority:    in.Priority,
		AssigneeID:  in.AssigneeID,
		DueDate:     in.DueDate,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, created)
}

// List handles GET /workspaces/{workspaceID}/projects/{projectID}/tasks.
//
// Query parameters (all optional): status_id, assignee_id (an id, or "none"),
// priority, q (search in titles), sort (position, newest, oldest, due),
// limit (1 to 200, default 100), offset.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.project(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	f := repository.TaskFilter{
		StatusID:   query.Get("status_id"),
		AssigneeID: query.Get("assignee_id"),
		Priority:   query.Get("priority"),
		Query:      strings.TrimSpace(query.Get("q")),
		Sort:       query.Get("sort"),
		Limit:      100,
	}

	v := validate.New()
	v.Check(f.StatusID == "" || validate.UUID(f.StatusID), "status_id", "status_id must be a valid id")
	v.Check(f.AssigneeID == "" || f.AssigneeID == "none" || validate.UUID(f.AssigneeID), "assignee_id",
		`assignee_id must be a valid id, or "none"`)
	v.Check(f.Priority == "" || validate.OneOf(f.Priority, priorities...), "priority",
		"priority must be one of: low, medium, high, urgent")
	v.Check(validate.MaxLen(f.Query, 100), "q", "q must be at most 100 characters")
	v.Check(f.Sort == "" || validate.OneOf(f.Sort, repository.TaskSorts()...), "sort",
		"sort must be one of: "+strings.Join(repository.TaskSorts(), ", "))
	if raw := query.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		v.Check(err == nil && n >= 1 && n <= 200, "limit", "limit must be a number from 1 to 200")
		if err == nil {
			f.Limit = n
		}
	}
	if raw := query.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		v.Check(err == nil && n >= 0, "offset", "offset must be 0 or more")
		if err == nil {
			f.Offset = n
		}
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	tasks, total, err := h.tasks.List(r.Context(), actor.WorkspaceID, projectID, f)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.DataMeta(w, http.StatusOK, tasks, map[string]int{"total": total, "limit": f.Limit, "offset": f.Offset})
}

// Get handles GET .../tasks/{number}.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}

	t, err := h.tasks.Get(r.Context(), actor.WorkspaceID, projectID, number)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, t)
}

// updateTaskRequest uses Optional so that a missing field (leave it alone) can
// be told apart from null (clear it).
type updateTaskRequest struct {
	Title       request.Optional[string] `json:"title"`
	Description request.Optional[string] `json:"description"`
	Priority    request.Optional[string] `json:"priority"`
	AssigneeID  request.Optional[string] `json:"assignee_id"`
	DueDate     request.Optional[string] `json:"due_date"`
}

// Update handles PATCH .../tasks/{number}: change only the fields that were sent.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}

	var in updateTaskRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}

	v := validate.New()
	var patch repository.TaskPatch

	if in.Title.Set {
		if in.Title.Value == nil {
			v.Check(false, "title", "title cannot be cleared")
		} else {
			title := strings.TrimSpace(*in.Title.Value)
			checkTitle(v, title)
			patch.Title = &title
		}
	}
	if in.Description.Set {
		if in.Description.Value == nil {
			v.Check(false, "description", "description cannot be null: use an empty string")
		} else {
			description := strings.TrimSpace(*in.Description.Value)
			checkDescription(v, description)
			patch.Description = &description
		}
	}
	if in.Priority.Set {
		if in.Priority.Value == nil || !validate.OneOf(*in.Priority.Value, priorities...) {
			v.Check(false, "priority", "priority must be one of: low, medium, high, urgent")
		} else {
			patch.Priority = in.Priority.Value
		}
	}
	if in.AssigneeID.Set {
		if in.AssigneeID.Value != nil {
			v.Check(validate.UUID(*in.AssigneeID.Value), "assignee_id", "assignee_id must be a valid id, or null to unassign")
		}
		patch.SetAssignee, patch.AssigneeID = true, in.AssigneeID.Value
	}
	if in.DueDate.Set {
		if in.DueDate.Value != nil {
			v.Check(validDate(*in.DueDate.Value), "due_date", "due_date must look like 2026-10-20, or null to clear it")
		}
		patch.SetDueDate, patch.DueDate = true, in.DueDate.Value
	}
	if patch.Empty() && v.Valid() {
		v.Check(false, "body", "send at least one field to change")
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	updated, err := h.tasks.Update(r.Context(), actor.WorkspaceID, projectID, number, patch)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, updated)
}

type moveTaskRequest struct {
	StatusID string `json:"status_id"`
	Before   *int   `json:"before"` // put the task directly above the task with this number; null means the bottom
}

// Move handles POST .../tasks/{number}/move: change a task's column and its place in it.
func (h *TaskHandler) Move(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}

	var in moveTaskRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}

	v := validate.New()
	v.Check(validate.UUID(in.StatusID), "status_id", "status_id must be a valid id")
	if in.Before != nil {
		v.Check(*in.Before >= 1, "before", "before must be a task number")
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	moved, err := h.tasks.Move(r.Context(), actor.WorkspaceID, projectID, number, in.StatusID, in.Before)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, moved)
}

// Delete handles DELETE .../tasks/{number}. The task is hidden, not erased.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}

	if err := h.tasks.Delete(r.Context(), actor.WorkspaceID, projectID, number); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// project finds who is asking and which project they mean.
func (h *TaskHandler) project(w http.ResponseWriter, r *http.Request) (services.Actor, string, bool) {
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

// task also reads the task number from the URL. A malformed number is
// answered like an unknown task.
func (h *TaskHandler) task(w http.ResponseWriter, r *http.Request) (services.Actor, string, int, bool) {
	actor, projectID, ok := h.project(w, r)
	if !ok {
		return services.Actor{}, "", 0, false
	}

	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil || number < 1 {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
		return services.Actor{}, "", 0, false
	}
	return actor, projectID, number, true
}

func checkTitle(v *validate.Validator, title string) {
	v.Check(validate.NotBlank(title), "title", "title is required")
	v.Check(validate.MaxLen(title, 200), "title", "title must be at most 200 characters")
}

func checkDescription(v *validate.Validator, description string) {
	v.Check(validate.MaxLen(description, 10000), "description", "description must be at most 10000 characters")
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func (h *TaskHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "project not found")
	case errors.Is(err, services.ErrTaskNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
	case errors.Is(err, services.ErrInvalidStatus):
		response.ValidationError(w, map[string]string{"status_id": "must be one of this project's columns"})
	case errors.Is(err, services.ErrAssigneeNotMember):
		response.ValidationError(w, map[string]string{"assignee_id": "must be a member of this workspace"})
	case errors.Is(err, services.ErrInvalidNeighbour):
		response.ValidationError(w, map[string]string{"before": "must be a task in the target column"})
	default:
		h.logger.Error("task request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
