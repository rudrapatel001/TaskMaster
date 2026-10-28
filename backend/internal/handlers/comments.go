package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/request"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

const maxCommentLength = 5000

type CommentHandler struct {
	comments *services.CommentService
	logger   *slog.Logger
}

func NewCommentHandler(comments *services.CommentService, logger *slog.Logger) *CommentHandler {
	return &CommentHandler{comments: comments, logger: logger}
}

type commentRequest struct {
	Body string `json:"body"`
}

// List handles GET .../tasks/{number}/comments.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}

	comments, err := h.comments.List(r.Context(), actor.WorkspaceID, projectID, number)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, comments)
}

// Create handles POST .../tasks/{number}/comments.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}

	created, err := h.comments.Create(r.Context(), actor, projectID, number, body)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, created)
}

// Update handles PATCH .../tasks/{number}/comments/{commentID}.
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}
	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}

	updated, err := h.comments.Update(r.Context(), actor, projectID, number, commentID, body)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, updated)
}

// Delete handles DELETE .../tasks/{number}/comments/{commentID}.
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, projectID, number, ok := h.task(w, r)
	if !ok {
		return
	}
	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}

	if err := h.comments.Delete(r.Context(), actor, projectID, number, commentID); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// readBody reads and checks the comment text. It answers the request itself
// when something is wrong.
func (h *CommentHandler) readBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in commentRequest
	if !request.DecodeJSON(w, r, &in) {
		return "", false
	}
	body := strings.TrimSpace(in.Body)

	v := validate.New()
	v.Check(validate.NotBlank(body), "body", "a comment cannot be empty")
	v.Check(validate.MaxLen(body, maxCommentLength), "body", "a comment can be at most 5000 characters")
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return "", false
	}
	return body, true
}

// task finds who is asking and which task they mean. Malformed ids and
// numbers are answered like unknown ones.
func (h *CommentHandler) task(w http.ResponseWriter, r *http.Request) (services.Actor, string, int, bool) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return services.Actor{}, "", 0, false
	}

	projectID := chi.URLParam(r, "projectID")
	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if !validate.UUID(projectID) || err != nil || number < 1 {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
		return services.Actor{}, "", 0, false
	}
	return actor, projectID, number, true
}

func (h *CommentHandler) commentID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "commentID")
	if !validate.UUID(id) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "comment not found")
		return "", false
	}
	return id, true
}

func (h *CommentHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrTaskNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "task not found")
	case errors.Is(err, services.ErrCommentNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "comment not found")
	case errors.Is(err, services.ErrForbidden):
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "you can only change your own comments")
	default:
		h.logger.Error("comment request failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
	}
}
