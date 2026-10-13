package permissions

import (
	"testing"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

const (
	viewer = models.RoleViewer
	member = models.RoleMember
	admin  = models.RoleAdmin
	owner  = models.RoleOwner
)

var allRoles = []models.Role{viewer, member, admin, owner}

func contains(list []models.Role, r models.Role) bool {
	for _, x := range list {
		if x == r {
			return true
		}
	}
	return false
}

func TestCanChangeRole(t *testing.T) {
	// For each actor: which targets they may touch, and which roles they may hand out.
	// (Viewers and members may not manage anyone, so their lists are empty.)
	type rule struct{ targets, grants []models.Role }
	rules := map[models.Role]rule{
		viewer: {},
		member: {},
		admin:  {targets: []models.Role{viewer, member}, grants: []models.Role{viewer, member}},
		owner:  {targets: allRoles, grants: allRoles},
	}

	for _, actor := range allRoles {
		for _, target := range allRoles {
			for _, newRole := range allRoles {
				want := contains(rules[actor].targets, target) && contains(rules[actor].grants, newRole)
				if got := CanChangeRole(actor, target, newRole); got != want {
					t.Errorf("CanChangeRole(actor=%s, target=%s, newRole=%s) = %v, want %v", actor, target, newRole, got, want)
				}
			}
		}
	}
}

func TestCanRemove(t *testing.T) {
	targets := map[models.Role][]models.Role{
		viewer: nil,
		member: nil,
		admin:  {viewer, member},
		owner:  allRoles,
	}
	for _, actor := range allRoles {
		for _, target := range allRoles {
			want := contains(targets[actor], target)
			if got := CanRemove(actor, target); got != want {
				t.Errorf("CanRemove(actor=%s, target=%s) = %v, want %v", actor, target, got, want)
			}
		}
	}
}

func TestMemberRulesRejectUnknownRoles(t *testing.T) {
	bogus := models.Role("superadmin")
	if CanChangeRole(owner, member, bogus) || CanChangeRole(owner, bogus, member) || CanChangeRole(bogus, member, viewer) {
		t.Error("unknown roles must never be accepted")
	}
	if CanRemove(owner, bogus) || CanRemove(bogus, member) {
		t.Error("unknown roles must never be accepted")
	}
}

func TestCanInvite(t *testing.T) {
	// For each actor: the roles they may invite. Nobody may invite an owner.
	grants := map[models.Role][]models.Role{
		viewer: nil,
		member: nil,
		admin:  {viewer, member},
		owner:  {viewer, member, admin},
	}
	for _, actor := range allRoles {
		for _, role := range allRoles {
			want := contains(grants[actor], role)
			if got := CanInvite(actor, role); got != want {
				t.Errorf("CanInvite(actor=%s, role=%s) = %v, want %v", actor, role, got, want)
			}
		}
	}
	if CanInvite(owner, models.Role("superadmin")) {
		t.Error("an unknown role must never be accepted")
	}
}
