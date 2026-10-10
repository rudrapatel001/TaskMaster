package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/httprate"

	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// RateLimit allows each client at most limit requests per window. Beyond that
// it answers 429 until the window passes.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(limit, window, clientKey,
		httprate.WithLimitHandler(func(w http.ResponseWriter, _ *http.Request) {
			response.Error(w, http.StatusTooManyRequests, response.CodeRateLimited,
				"too many requests, please try again later")
		}),
	)
}

// clientKey identifies a client by the address of the network connection.
// Headers such as X-Forwarded-For are NOT used, because clients can fake them
// to dodge the limit.
//
// Note: behind a proxy or load balancer, every client shares the proxy's
// address, so this must be revisited when deploying.
func clientKey(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return httprate.CanonicalizeIP(host), nil
}
