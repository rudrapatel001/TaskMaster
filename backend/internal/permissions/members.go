package permissions

import "github.com/rudrapatel001/taskmaster/backend/internal/models"

// These two functions are the rules about people that the permission table
// cannot express: they depend on WHO is being managed, not just on what the
// actor may do in general.
//
//   - Owners can manage anyone.
//   - Everyone else with the "manage members" permission (admins) can only deal
//     with people ranked below them, and can only hand out roles below their own.
//
// The "a workspace must keep at least one owner" rule needs the database, so it
// is enforced where the change is made (see the repository).

// CanChangeRole reports whether actor may change a member whose current role
// is target into newRole.
func CanChangeRole(actor, target, newRole models.Role) bool {
	if !Can(actor, MembersManage) || !target.Valid() || !newRole.Valid() {
		return false
	}
	if actor == models.RoleOwner {
		return true
	}
	return target.Rank() < actor.Rank() && newRole.Rank() < actor.Rank()
}

// CanRemove reports whether actor may remove a member whose role is target.
func CanRemove(actor, target models.Role) bool {
	if !Can(actor, MembersManage) || !target.Valid() {
		return false
	}
	if actor == models.RoleOwner {
		return true
	}
	return target.Rank() < actor.Rank()
}

// CanInvite reports whether actor may invite someone with the given role.
// Nobody can be invited straight in as an owner: ownership is handed over by an
// existing owner, never through a link.
func CanInvite(actor, role models.Role) bool {
	if !Can(actor, MembersInvite) || !role.Valid() || role == models.RoleOwner {
		return false
	}
	if actor == models.RoleOwner {
		return true
	}
	return role.Rank() < actor.Rank()
}
