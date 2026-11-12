package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type SearchHandler struct {
	search *services.SearchService
	logger *slog.Logger
}

func NewSearchHandler(search *services.SearchService, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{search: search, logger: logger}
}

// Search handles GET /workspaces/{workspaceID}/search?q=...&limit=...
//
// q is required (2 to 100 characters). limit is optional (1 to 20, default 8).
// The answer is {"data": {"tasks": [...]}}: tasks are the only kind of result
// for now, but the object leaves room for more.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(r)
	if !ok {
		forbidden(w)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 8
	v := validate.New()
	v.Check(validate.MinLen(q, 2), "q", "q must be at least 2 characters")
	v.Check(validate.MaxLen(q, 100), "q", "q must be at most 100 characters")
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		v.Check(err == nil && n >= 1 && n <= 20, "limit", "limit must be a number from 1 to 20")
		if err == nil {
			limit = n
		}
	}
	if !v.Valid() {
		response.ValidationError(w, v.Fields())
		return
	}

	tasks, err := h.search.Tasks(r.Context(), actor.WorkspaceID, q, limit)
	if err != nil {
		h.logger.Error("search failed",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"error", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
		return
	}

	response.Data(w, http.StatusOK, map[string]any{"tasks": tasks})
}
