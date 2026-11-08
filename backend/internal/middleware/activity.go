package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// TaskActivityRecorder receives task events. It is satisfied by the object that
// feeds both the activity feed and notifications (services.TaskEvents).
type TaskActivityRecorder interface {
	RecordTask(ctx context.Context, e models.TaskEvent)
}

// RecordTaskActivity wraps a route that changes a task. When the request
// succeeds (a 2xx answer), it reports an event with the given action, such as
// "task.moved". Failed requests report nothing.
//
// It runs after the handler, so it must sit inside Authenticate,
// WorkspaceAccess and RequirePermission.
func RecordTaskActivity(recorder TaskActivityRecorder, action string) func(http.Handler) http.Handler {
	// The response is kept when it holds something we need: a new task's number
	// (it is not in the URL), an edited task's assignee, a comment's text.
	needsResponse := action == "task.created" || action == "task.updated" || action == "task.commented"
	// For edits, which fields the request tried to change.
	needsRequest := action == "task.updated"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var sentFields []string
			if needsRequest {
				sentFields = readFieldNames(r)
			}

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

			// What the response says: the task's number and assignee, or a comment's text.
			var sent struct {
				Data struct {
					Number     int     `json:"number"`
					Body       string  `json:"body"`
					AssigneeID *string `json:"assignee_id"`
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
				Fields:      sentFields,
				AssigneeID:  sent.Data.AssigneeID,
			}
			for _, field := range sentFields {
				if field == "assignee_id" {
					event.AssigneeSent = true
				}
			}
			if action == "task.commented" {
				event.Extra = map[string]any{"excerpt": excerpt(sent.Data.Body, 100)}
			}

			// WithoutCancel: if the browser has already gone, still write the entry.
			recorder.RecordTask(context.WithoutCancel(r.Context()), event)
		})
	}
}

// readFieldNames returns the names of the fields in a JSON request body, and
// puts the body back so the handler reads exactly the same bytes. At most 1 MiB
// is looked at; anything bigger is left for the handler to refuse.
func readFieldNames(r *http.Request) []string {
	const limit = 1 << 20
	buf, _ := io.ReadAll(io.LimitReader(r.Body, limit+1))
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), r.Body))
	if len(buf) > limit {
		return nil
	}

	var fields map[string]json.RawMessage
	if json.Unmarshal(buf, &fields) != nil {
		return nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// excerpt shortens text to at most max characters, adding an ellipsis if cut.
func excerpt(text string, max int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}
