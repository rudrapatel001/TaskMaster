package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"
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

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	if err := h.DB.Ping(ctx); err != nil {
		h.Logger.Error("health check: database ping failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable","database":"down"}`))
		return
	}

	_, _ = w.Write([]byte(`{"status":"ok","database":"up"}`))
}
