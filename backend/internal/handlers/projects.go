package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

// A project key: 2 to 10 capital letters or digits, starting with a letter.
var projectKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

type ProjectHandler struct {
	projects *services.ProjectService
	logger   *slog.Logger
}

func NewProjectHandler(projects *services.ProjectService, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{projects: projects, logger: logger}
}

type createProjectRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create handles POST /workspaces/{workspaceID}/projects.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	var in createProjectRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Key = strings.ToUpper(strings.TrimSpace(in.Key))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	v := validate.New()
	v.Check(validate.NotBlank(in.Key), "key", "key is required")
	v.Check(validate.Matches(in.Key, projectKeyPattern), "key",
		"key must be 2 to 10 characters: capital letters and digits, starting with a letter")
	checkProjectText(v, in.Name, in.Description)
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	created, err := h.projects.Create(r.Context(), actor.WorkspaceID, actor.UserID, in.Key, in.Name, in.Description)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, created)
}

// List handles GET /workspaces/{workspaceID}/projects.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	projects, err := h.projects.List(r.Context(), actor.WorkspaceID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, projects)
}

// Get handles GET /workspaces/{workspaceID}/projects/{projectID}.
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.target(w, r)
	if !ok {
		return
	}

	project, err := h.projects.Get(r.Context(), actor.WorkspaceID, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, project)
}

// Update handles PATCH /workspaces/{workspaceID}/projects/{projectID}.
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.target(w, r)
	if !ok {
		return
	}

	var in updateProjectRequest
	if !request.DecodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	v := validate.New()
	checkProjectText(v, in.Name, in.Description)
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	updated, err := h.projects.Update(r.Context(), actor.WorkspaceID, projectID, in.Name, in.Description)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, updated)
}

// Delete handles DELETE /workspaces/{workspaceID}/projects/{projectID}.
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, projectID, ok := h.target(w, r)
	if !ok {
		return
	}

	if err := h.projects.Delete(r.Context(), actor.WorkspaceID, projectID); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// target finds who is asking and which project they mean. A malformed project
// id is answered like an unknown one.
func (h *ProjectHandler) target(w http.ResponseWriter, r *http.Request) (services.Actor, string, bool) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return services.Actor{}, "", false
	}

	projectID := chi.URLParam(r, "projectID")
	if !validate.UUID(projectID) {
		h.notFound(w)
		return services.Actor{}, "", false
	}
	return actor, projectID, true
}

func checkProjectText(v *validate.Validator, name, description string) {
	v.Check(validate.NotBlank(name), "name", "name is required")
	v.Check(validate.MaxLen(name, 100), "name", "name must be at most 100 characters")
	v.Check(validate.MaxLen(description, 2000), "description", "description must be at most 2000 characters")
}

func (h *ProjectHandler) notFound(w http.ResponseWriter) {
	response.Error(w, http.StatusNotFound, response.CodeNotFound, "project not found")
}

func (h *ProjectHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		h.notFound(w)
	case errors.Is(err, services.ErrProjectKeyTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict,
			"a project with this key already exists in the workspace")
	default:
		h.logger.Error("project request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
