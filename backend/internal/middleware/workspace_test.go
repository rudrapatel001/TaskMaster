package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
)

const wsID = "3567b73b-548b-466d-8426-95ba419ddf1b"

// fakeLoader knows one workspace and the roles of a few users in it.
type fakeLoader struct {
	roles map[string]models.Role // user id -> role in wsID
	err   error                  // if set, every lookup fails with this
	calls int
}

func (f *fakeLoader) GetMembership(_ context.Context, workspaceID, userID string) (*models.WorkspaceMembership, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	role, ok := f.roles[userID]
	// PostgreSQL compares uuid text without regard to capitalization, so the fake does too.
	if !ok || !strings.EqualFold(workspaceID, wsID) {
		return nil, services.ErrWorkspaceNotFound
	}
	return &models.WorkspaceMembership{Workspace: models.Workspace{ID: workspaceID, Name: "WS"}, Role: role}, nil
}

// newTestRouter wires the middleware the same way the real router does. The
// "login" step is faked: the X-User header becomes the user id in the context.
func newTestRouter(loader MembershipLoader) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if u := req.Header.Get("X-User"); u != "" {
				req = req.WithContext(context.WithValue(req.Context(), userIDKey{}, u))
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Route("/workspaces/{workspaceID}", func(r chi.Router) {
		r.Use(WorkspaceAccess(loader, slog.New(slog.NewTextHandler(io.Discard, nil))))
		ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }
		r.With(RequirePermission(permissions.WorkspaceView)).Get("/", ok)
		r.With(RequirePermission(permissions.WorkspaceUpdate)).Patch("/", ok)
		r.With(RequirePermission(permissions.WorkspaceDelete)).Delete("/", ok)
		r.With(RequirePermission(permissions.MembersView)).Get("/members", ok)
		r.With(RequirePermission(permissions.Permission("made:up"))).Get("/secret", ok)
	})
	return r
}

func do(h http.Handler, method, path, user string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if user != "" {
		req.Header.Set("X-User", user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestEveryRoleAgainstEveryRoute(t *testing.T) {
	loader := &fakeLoader{roles: map[string]models.Role{
		"viewer": models.RoleViewer, "member": models.RoleMember, "admin": models.RoleAdmin, "owner": models.RoleOwner,
	}}
	h := newTestRouter(loader)

	routes := []struct{ method, path string }{
		{"GET", "/workspaces/" + wsID},
		{"PATCH", "/workspaces/" + wsID},
		{"DELETE", "/workspaces/" + wsID},
		{"GET", "/workspaces/" + wsID + "/members"},
	}
	// expected status for each route, in the order viewer, member, admin, owner
	want := [][]int{
		{200, 200, 200, 200}, // view
		{403, 403, 200, 200}, // rename
		{403, 403, 403, 200}, // delete
		{200, 200, 200, 200}, // members
	}
	for i, route := range routes {
		for j, user := range []string{"viewer", "member", "admin", "owner"} {
			got := do(h, route.method, route.path, user).Code
			if got != want[i][j] {
				t.Errorf("%s %s as %s: got %d, want %d", route.method, route.path, user, got, want[i][j])
			}
		}
		// someone who is not a member always gets 404, whatever the route
		if got := do(h, route.method, route.path, "stranger").Code; got != 404 {
			t.Errorf("%s %s as a non-member: got %d, want 404", route.method, route.path, got)
		}
	}
}

func TestNonMemberCannotTellWhetherAWorkspaceExists(t *testing.T) {
	h := newTestRouter(&fakeLoader{roles: map[string]models.Role{"member": models.RoleMember}})
	realButNotMine := do(h, "GET", "/workspaces/"+wsID, "stranger")
	doesNotExist := do(h, "GET", "/workspaces/11111111-1111-1111-1111-111111111111", "stranger")
	if realButNotMine.Code != doesNotExist.Code || realButNotMine.Code != 404 {
		t.Fatalf("statuses differ: %d vs %d", realButNotMine.Code, doesNotExist.Code)
	}
	if realButNotMine.Body.String() != doesNotExist.Body.String() {
		t.Errorf("bodies differ:\n%s\n%s", realButNotMine.Body, doesNotExist.Body)
	}
}

func TestMalformedWorkspaceIDsNeverReachTheDatabase(t *testing.T) {
	loader := &fakeLoader{roles: map[string]models.Role{"owner": models.RoleOwner}}
	h := newTestRouter(loader)
	for _, bad := range []string{"abc", "123", "not-a-uuid", wsID + "x", "' OR 1=1 --", "00000000-0000-0000-0000-00000000000g"} {
		// browsers URL-encode odd characters, so the test does too
		if got := do(h, "GET", "/workspaces/"+url.PathEscape(bad), "owner").Code; got != 404 {
			t.Errorf("id %q: got %d, want 404", bad, got)
		}
	}
	if loader.calls != 0 {
		t.Errorf("malformed ids caused %d database lookups, want 0", loader.calls)
	}
	// an uppercase uuid is still a valid uuid
	if got := do(h, "GET", "/workspaces/"+strings.ToUpper(wsID), "owner").Code; got != 200 {
		t.Errorf("uppercase uuid: got %d, want 200", got)
	}
}

func TestDatabaseFailureIsA500NotAnAccidentalAllow(t *testing.T) {
	h := newTestRouter(&fakeLoader{err: errors.New("database is down")})
	if got := do(h, "GET", "/workspaces/"+wsID, "owner").Code; got != 500 {
		t.Errorf("got %d, want 500", got)
	}
}

func TestNoLoginMeans401(t *testing.T) {
	h := newTestRouter(&fakeLoader{roles: map[string]models.Role{"owner": models.RoleOwner}})
	if got := do(h, "GET", "/workspaces/"+wsID, "").Code; got != 401 {
		t.Errorf("got %d, want 401", got)
	}
}

func TestUnknownPermissionIsDeniedEvenForOwner(t *testing.T) {
	h := newTestRouter(&fakeLoader{roles: map[string]models.Role{"owner": models.RoleOwner}})
	if got := do(h, "GET", "/workspaces/"+wsID+"/secret", "owner").Code; got != 403 {
		t.Errorf("got %d, want 403", got)
	}
}

func TestRequirePermissionFailsClosedWithoutMembership(t *testing.T) {
	// RequirePermission used by mistake WITHOUT WorkspaceAccess in front of it
	r := chi.NewRouter()
	r.With(RequirePermission(permissions.WorkspaceView)).Get("/x", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	if got := do(r, "GET", "/x", "owner").Code; got != 403 {
		t.Errorf("got %d, want 403 (it must refuse when no membership was loaded)", got)
	}
}
