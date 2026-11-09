package models

// Role is what a person is allowed to be inside one workspace.
type Role string

const (
	RoleViewer Role = "viewer"
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
)

// Rank orders the roles: a higher number means more power.
// Anything that is not a real role has rank 0, so it never outranks anyone.
func (r Role) Rank() int {
	switch r {
	case RoleViewer:
		return 1
	case RoleMember:
		return 2
	case RoleAdmin:
		return 3
	case RoleOwner:
		return 4
	default:
		return 0
	}
}

// Valid reports whether r is one of the four real roles.
func (r Role) Valid() bool {
	return r.Rank() > 0
}
