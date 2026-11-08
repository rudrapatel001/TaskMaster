package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
	"github.com/rudrapatel001/taskmaster/backend/internal/validate"
)

type NotificationHandler struct {
	notifications *services.NotificationService
	logger        *slog.Logger
}

func NewNotificationHandler(notifications *services.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{notifications: notifications, logger: logger}
}

// List handles GET /notifications. Query parameters (optional): limit (1 to
// 100, default 20) and before (the next_cursor of the previous page).
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			response.ValidationError(w, map[string]string{"limit": "limit must be a number from 1 to 100"})
			return
		}
		limit = n
	}

	page, err := h.notifications.List(r.Context(), userID, limit, r.URL.Query().Get("before"))
	if err != nil {
		if errors.Is(err, services.ErrInvalidCursor) {
			response.ValidationError(w, map[string]string{"before": "before is not a valid cursor"})
			return
		}
		h.fail(w, r, err)
		return
	}

	response.DataMeta(w, http.StatusOK, page.Notifications, map[string]any{"next_cursor": page.NextCursor})
}

// UnreadCount handles GET /notifications/unread-count: the number on the bell.
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}

	count, err := h.notifications.UnreadCount(r.Context(), userID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, map[string]int{"count": count})
}

// MarkRead handles POST /notifications/{notificationID}/read.
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "notificationID")
	if !validate.UUID(id) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "notification not found")
		return
	}

	if err := h.notifications.MarkRead(r.Context(), userID, id); err != nil {
		if errors.Is(err, services.ErrNotificationNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "notification not found")
			return
		}
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead handles POST /notifications/read-all.
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}

	if err := h.notifications.MarkAllRead(r.Context(), userID); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// user reads who is logged in. Notifications belong to a person, not to a
// workspace, so there is no membership to look at.
func (h *NotificationHandler) user(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid access token")
		return "", false
	}
	return userID, true
}

func (h *NotificationHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("notification request failed",
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"error", err)
	response.Error(w, http.StatusInternalServerError, response.CodeInternal, "something went wrong")
}
