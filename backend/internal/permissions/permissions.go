package permissions

import "github.com/rudrapatel001/taskmaster/backend/internal/models"

// Permission is one specific thing a person might be allowed to do.
type Permission string

const (
	WorkspaceView   Permission = "workspace:view"
	WorkspaceUpdate Permission = "workspace:update"
	WorkspaceDelete Permission = "workspace:delete"
	WorkspaceLeave  Permission = "workspace:leave"

	MembersView   Permission = "members:view"
	MembersInvite Permission = "members:invite"
	MembersManage Permission = "members:manage"

	ProjectView   Permission = "project:view"
	ProjectManage Permission = "project:manage"

	TaskView  Permission = "task:view"
	TaskWrite Permission = "task:write"

	CommentWrite    Permission = "comment:write"
	CommentModerate Permission = "comment:moderate"
)

// minimumRole is the whole permission table: for each permission, the lowest
// role that has it. Because every role includes everything the roles below it
// can do, this one fact per permission is enough.
var minimumRole = map[Permission]models.Role{
	WorkspaceView:   models.RoleViewer,
	WorkspaceUpdate: models.RoleAdmin,
	WorkspaceDelete: models.RoleOwner,
	WorkspaceLeave:  models.RoleViewer,

	MembersView:   models.RoleViewer,
	MembersInvite: models.RoleAdmin,
	MembersManage: models.RoleAdmin,

	ProjectView:   models.RoleViewer,
	ProjectManage: models.RoleAdmin,

	TaskView:  models.RoleViewer,
	TaskWrite: models.RoleMember,

	CommentWrite:    models.RoleMember,
	CommentModerate: models.RoleAdmin,
}

// Can reports whether someone with the given role may do the given thing.
// It denies by default: an unknown permission or an unknown role is refused.
func Can(role models.Role, permission Permission) bool {
	required, known := minimumRole[permission]
	if !known {
		return false
	}
	return role.Rank() > 0 && role.Rank() >= required.Rank()
}
