package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/rudrapatel001/taskmaster/backend/internal/handlers"
	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
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
	Workspaces     *handlers.WorkspaceHandler
	Invitations    *handlers.InvitationHandler
	Projects       *handlers.ProjectHandler
	Tasks          *handlers.TaskHandler
	Comments       *handlers.CommentHandler
	Activity       *handlers.ActivityHandler

	// ActivityRecorder writes entries to the activity feed when task routes succeed.
	ActivityRecorder middleware.TaskActivityRecorder

	// Authenticate protects routes that need a login.
	Authenticate func(http.Handler) http.Handler
	// WorkspaceAccess protects routes inside one workspace: it loads the
	// caller's membership, or answers 404.
	WorkspaceAccess func(http.Handler) http.Handler
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

	// record turns on activity recording for a route (see middleware/activity.go):
	// after the route succeeds, an entry with this action is added to the feed.
	record := func(action string) func(http.Handler) http.Handler {
		return middleware.RecordTaskActivity(d.ActivityRecorder, action)
	}

	r.Get("/health", d.Health.Check)

	// Anyone holding an invitation link may see what it is for, before logging in.
	r.With(middleware.RateLimit(30, time.Minute)).Get("/invitations/{token}", d.Invitations.Preview)

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

		// Accepting needs a login: the invitation must match your account's email.
		r.With(middleware.RateLimit(10, time.Minute)).Post("/invitations/{token}/accept", d.Invitations.Accept)

		r.Post("/workspaces", d.Workspaces.Create)
		r.Get("/workspaces", d.Workspaces.List)

		// Everything inside one workspace: first check the caller is a member
		// (WorkspaceAccess), then check their role allows the action
		// (RequirePermission).
		r.Route("/workspaces/{workspaceID}", func(r chi.Router) {
			r.Use(d.WorkspaceAccess)

			r.With(middleware.RequirePermission(permissions.WorkspaceView)).Get("/", d.Workspaces.Get)
			r.With(middleware.RequirePermission(permissions.WorkspaceUpdate)).Patch("/", d.Workspaces.Rename)
			r.With(middleware.RequirePermission(permissions.WorkspaceDelete)).Delete("/", d.Workspaces.Delete)

			r.With(middleware.RequirePermission(permissions.MembersView)).Get("/members", d.Workspaces.ListMembers)
			r.With(middleware.RequirePermission(permissions.MembersManage)).Patch("/members/{userID}", d.Workspaces.ChangeMemberRole)
			r.With(middleware.RequirePermission(permissions.MembersManage)).Delete("/members/{userID}", d.Workspaces.RemoveMember)
			r.With(middleware.RequirePermission(permissions.WorkspaceLeave)).Post("/leave", d.Workspaces.Leave)

			r.With(middleware.RequirePermission(permissions.MembersInvite)).Get("/invitations", d.Invitations.List)
			r.With(middleware.RequirePermission(permissions.MembersInvite)).Post("/invitations", d.Invitations.Create)
			r.With(middleware.RequirePermission(permissions.MembersInvite)).Delete("/invitations/{invitationID}", d.Invitations.Revoke)

			// Everything that happened in this workspace, newest first.
			r.With(middleware.RequirePermission(permissions.WorkspaceView)).Get("/activity", d.Activity.Workspace)

			r.Route("/projects", func(r chi.Router) {
				r.With(middleware.RequirePermission(permissions.ProjectView)).Get("/", d.Projects.List)
				r.With(middleware.RequirePermission(permissions.ProjectManage)).Post("/", d.Projects.Create)

				r.Route("/{projectID}", func(r chi.Router) {
					r.With(middleware.RequirePermission(permissions.ProjectView)).Get("/", d.Projects.Get)
					r.With(middleware.RequirePermission(permissions.ProjectManage)).Patch("/", d.Projects.Update)
					r.With(middleware.RequirePermission(permissions.ProjectManage)).Delete("/", d.Projects.Delete)

					r.Route("/tasks", func(r chi.Router) {
						r.With(middleware.RequirePermission(permissions.TaskView)).Get("/", d.Tasks.List)
						r.With(middleware.RequirePermission(permissions.TaskWrite), record("task.created")).Post("/", d.Tasks.Create)

						r.Route("/{number}", func(r chi.Router) {
							r.With(middleware.RequirePermission(permissions.TaskView)).Get("/", d.Tasks.Get)
							r.With(middleware.RequirePermission(permissions.TaskWrite), record("task.updated")).Patch("/", d.Tasks.Update)
							r.With(middleware.RequirePermission(permissions.TaskWrite), record("task.deleted")).Delete("/", d.Tasks.Delete)
							r.With(middleware.RequirePermission(permissions.TaskWrite), record("task.moved")).Post("/move", d.Tasks.Move)

							r.With(middleware.RequirePermission(permissions.TaskView)).Get("/comments", d.Comments.List)
							r.With(middleware.RequirePermission(permissions.CommentWrite), record("task.commented")).Post("/comments", d.Comments.Create)
							// Who may edit or delete depends on who wrote the comment, so the service decides.
							r.With(middleware.RequirePermission(permissions.TaskView)).Patch("/comments/{commentID}", d.Comments.Update)
							r.With(middleware.RequirePermission(permissions.TaskView)).Delete("/comments/{commentID}", d.Comments.Delete)

							r.With(middleware.RequirePermission(permissions.TaskView)).Get("/activity", d.Activity.Task)
						})
					})
				})
			})
		})
	})

	return r
}
