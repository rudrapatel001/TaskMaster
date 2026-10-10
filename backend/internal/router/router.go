package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/rudrapatel001/taskmaster/backend/internal/handlers"
	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// Deps is everything the router needs. As the app grows, new handlers are
// added here instead of making New take ever more arguments.
type Deps struct {
	Logger         *slog.Logger
	AllowedOrigins []string
	Health         *handlers.HealthHandler
	Auth           *handlers.AuthHandler
	Users          *handlers.UserHandler

	// Authenticate is the middleware that protects routes needing a login.
	Authenticate func(http.Handler) http.Handler
}

// New builds the HTTP router with all middleware and routes registered.
func New(d Deps) http.Handler {
	r := chi.NewRouter()

	// Order matters: the first one listed is the outermost layer.
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(d.Logger))
	r.Use(middleware.Recoverer(d.Logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Replace chi's plain-text 404 and 405 pages with our JSON error format.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "method not allowed")
	})

	r.Get("/health", d.Health.Check)

	// Each of these has its own counter. Registering and logging in are the
	// expensive ones (password hashing), so they get the strictest limits.
	r.Route("/auth", func(r chi.Router) {
		r.With(middleware.RateLimit(5, time.Minute)).Post("/register", d.Auth.Register)
		r.With(middleware.RateLimit(10, time.Minute)).Post("/login", d.Auth.Login)
		r.With(middleware.RateLimit(30, time.Minute)).Post("/refresh", d.Auth.Refresh)
		r.Post("/logout", d.Auth.Logout)
	})

	// Everything in this group requires a valid access token.
	r.Group(func(r chi.Router) {
		r.Use(d.Authenticate)
		r.Get("/me", d.Users.Me)
		r.Patch("/me", d.Users.UpdateMe)
	})

	return r
}
