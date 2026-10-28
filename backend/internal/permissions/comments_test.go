package permissions

import (
	"testing"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

func TestCommentRules(t *testing.T) {
	cases := []struct {
		role         models.Role
		author       bool
		edit, delete bool
	}{
		{models.RoleViewer, true, false, true}, // a demoted author can remove, but not rewrite, their comment
		{models.RoleViewer, false, false, false},
		{models.RoleMember, true, true, true},
		{models.RoleMember, false, false, false},
		{models.RoleAdmin, true, true, true},
		{models.RoleAdmin, false, false, true}, // moderation: delete, but never rewrite someone else's words
		{models.RoleOwner, true, true, true},
		{models.RoleOwner, false, false, true},
	}
	for _, c := range cases {
		if got := CanEditComment(c.role, c.author); got != c.edit {
			t.Errorf("CanEditComment(%s, author=%v) = %v, want %v", c.role, c.author, got, c.edit)
		}
		if got := CanDeleteComment(c.role, c.author); got != c.delete {
			t.Errorf("CanDeleteComment(%s, author=%v) = %v, want %v", c.role, c.author, got, c.delete)
		}
	}
}
