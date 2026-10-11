package permissions

import (
	"testing"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

func TestCan(t *testing.T) {
	const (
		V = models.RoleViewer
		M = models.RoleMember
		A = models.RoleAdmin
		O = models.RoleOwner
	)

	// For each permission: the roles that SHOULD have it, written out by hand.
	allowed := map[Permission][]models.Role{
		WorkspaceView:   {V, M, A, O},
		WorkspaceUpdate: {A, O},
		WorkspaceDelete: {O},
		MembersView:     {V, M, A, O},
		MembersInvite:   {A, O},
		MembersManage:   {A, O},
		ProjectView:     {V, M, A, O},
		ProjectManage:   {A, O},
		TaskView:        {V, M, A, O},
		TaskWrite:       {M, A, O},
		CommentWrite:    {M, A, O},
		CommentModerate: {A, O},
	}

	if len(allowed) != len(minimumRole) {
		t.Fatalf("test lists %d permissions but the code defines %d: update the test", len(allowed), len(minimumRole))
	}

	for permission, roles := range allowed {
		for _, role := range []models.Role{V, M, A, O} {
			want := false
			for _, r := range roles {
				if r == role {
					want = true
				}
			}
			if got := Can(role, permission); got != want {
				t.Errorf("Can(%s, %s) = %v, want %v", role, permission, got, want)
			}
		}
	}
}

func TestCanDeniesByDefault(t *testing.T) {
	if Can(models.RoleOwner, Permission("made:up")) {
		t.Error("an unknown permission must be denied, even for an owner")
	}
	for _, role := range []models.Role{"", "superadmin", "OWNER", "root"} {
		if Can(role, WorkspaceView) {
			t.Errorf("unknown role %q must be denied everything", role)
		}
	}
}
