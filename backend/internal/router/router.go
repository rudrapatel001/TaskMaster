package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/handlers"
	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// New builds the HTTP router with all middleware and routes registered.
func New(logger *slog.Logger, db handlers.Pinger) http.Handler {
	r := chi.NewRouter()

	// Order matters: the first one listed is the outermost layer.
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recoverer(logger))

	// Replace chi's plain-text 404 and 405 pages with our JSON error format.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "method not allowed")
	})

	health := &handlers.HealthHandler{DB: db, Logger: logger}
	r.Get("/health", health.Check)

	return r
}
