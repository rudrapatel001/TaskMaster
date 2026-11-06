package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// TaskActivityRecorder writes feed entries. *services.ActivityService satisfies it.
type TaskActivityRecorder interface {
	RecordTask(ctx context.Context, e models.TaskEvent)
}

// RecordTaskActivity wraps a route that changes a task. When the request
// succeeds (a 2xx answer), it adds an entry with the given action, such as
// "task.moved", to the activity feed. Failed requests record nothing.
//
// It runs after the handler, so it must sit inside Authenticate,
// WorkspaceAccess and RequirePermission.
func RecordTaskActivity(recorder TaskActivityRecorder, action string) func(http.Handler) http.Handler {
	// Creating a task has no number in its URL, and a comment's text is wanted
	// for the entry, so for those two the response body is kept for reading.
	needsResponse := action == "task.created" || action == "task.commented"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			var response bytes.Buffer
			if needsResponse {
				ww.Tee(&response)
			}

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			if status < 200 || status > 299 {
				return
			}

			userID, hasUser := UserIDFromContext(r.Context())
			membership, hasMembership := MembershipFromContext(r.Context())
			if !hasUser || !hasMembership {
				return
			}

			// What the response says: the new task's number, or the comment's text.
			var sent struct {
				Data struct {
					Number int    `json:"number"`
					Body   string `json:"body"`
				} `json:"data"`
			}
			if needsResponse {
				_ = json.Unmarshal(response.Bytes(), &sent)
			}

			number, err := strconv.Atoi(chi.URLParam(r, "number"))
			if err != nil {
				number = sent.Data.Number // creating a task: the number is in the answer
			}
			if number < 1 {
				return
			}

			event := models.TaskEvent{
				WorkspaceID: membership.ID,
				ActorID:     userID,
				ProjectID:   chi.URLParam(r, "projectID"),
				Number:      number,
				Action:      action,
			}
			if action == "task.commented" {
				event.Extra = map[string]any{"excerpt": excerpt(sent.Data.Body, 100)}
			}

			// WithoutCancel: if the browser has already gone, still write the entry.
			recorder.RecordTask(context.WithoutCancel(r.Context()), event)
		})
	}
}

// excerpt shortens text to at most max characters, adding an ellipsis if cut.
func excerpt(text string, max int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}
