package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// Recoverer turns a panic inside a handler into a logged error and a clean
// 500 response, instead of a dropped connection.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// net/http uses this special panic to abort a response on purpose.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}

				logger.Error("panic recovered",
					"request_id", RequestIDFromContext(r.Context()),
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				response.Error(w, http.StatusInternalServerError,
					response.CodeInternal, "something went wrong")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
