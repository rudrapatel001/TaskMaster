package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/handlers"
)

// New builds the HTTP router with all routes registered.
func New(logger *slog.Logger, db handlers.Pinger) http.Handler {
	r := chi.NewRouter()

	health := &handlers.HealthHandler{DB: db, Logger: logger}
	r.Get("/health", health.Check)

	return r
}
