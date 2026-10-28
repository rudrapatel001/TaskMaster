package permissions

import "github.com/rudrapatel001/taskmaster/backend/internal/models"

// CanEditComment reports whether someone with this role may change the words
// of a comment. Only its author may, and only while their role still lets them
// comment (a viewer cannot write comments, so cannot rewrite them either).
func CanEditComment(role models.Role, isAuthor bool) bool {
	return isAuthor && Can(role, CommentWrite)
}

// CanDeleteComment reports whether someone with this role may delete a comment.
// Authors may always remove their own; admins and owners may remove anyone's
// (moderation).
func CanDeleteComment(role models.Role, isAuthor bool) bool {
	return isAuthor || Can(role, CommentModerate)
}