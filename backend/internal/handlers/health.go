package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// Pinger is anything that can check whether the database is reachable.
// *pgxpool.Pool satisfies it, and tests can pass a fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	DB     Pinger
	Logger *slog.Logger
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.DB.Ping(ctx); err != nil {
		h.Logger.Error("health check: database ping failed", "error", err)
		response.JSON(w, http.StatusServiceUnavailable,
			healthResponse{Status: "unavailable", Database: "down"})
		return
	}

	response.JSON(w, http.StatusOK, healthResponse{Status: "ok", Database: "up"})
}
