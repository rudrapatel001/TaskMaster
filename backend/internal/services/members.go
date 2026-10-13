package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

var (
	// ErrMemberNotFound means the person is not a member of the workspace.
	ErrMemberNotFound = errors.New("member not found")

	// ErrForbidden means the person's role does not allow this change.
	ErrForbidden = errors.New("not allowed")

	// ErrLastOwner means the change would leave the workspace without an owner.
	ErrLastOwner = errors.New("a workspace must keep at least one owner")

	// ErrMembershipChanged means someone else changed the membership at the same
	// moment. Trying again will give a clear answer.
	ErrMembershipChanged = errors.New("membership changed, please try again")
)

// Actor is the person performing an action inside one workspace.
type Actor struct {
	WorkspaceID string
	UserID      string
	Role        models.Role
}

// ChangeRole changes another member's role, if the actor's role allows it.
func (s *WorkspaceService) ChangeRole(ctx context.Context, actor Actor, targetID string, newRole models.Role) (*models.Member, error) {
	target, err := s.store.GetMember(ctx, actor.WorkspaceID, targetID)
	if err != nil {
		return nil, memberError(err)
	}

	if !permissions.CanChangeRole(actor.Role, target.Role, newRole) {
		return nil, ErrForbidden
	}
	if target.Role == newRole {
		return target, nil // nothing to change
	}

	updated, err := s.store.SetMemberRole(ctx, actor.WorkspaceID, targetID, target.Role, newRole)
	if err != nil {
		return nil, memberError(err)
	}
	return updated, nil
}

// RemoveMember removes another member, if the actor's role allows it.
func (s *WorkspaceService) RemoveMember(ctx context.Context, actor Actor, targetID string) error {
	target, err := s.store.GetMember(ctx, actor.WorkspaceID, targetID)
	if err != nil {
		return memberError(err)
	}

	if !permissions.CanRemove(actor.Role, target.Role) {
		return ErrForbidden
	}

	return memberError(s.store.RemoveMember(ctx, actor.WorkspaceID, targetID, target.Role))
}

// Leave removes the actor from the workspace. Anyone may leave, except the
// last owner.
func (s *WorkspaceService) Leave(ctx context.Context, actor Actor) error {
	return memberError(s.store.RemoveMember(ctx, actor.WorkspaceID, actor.UserID, actor.Role))
}

// memberError turns repository errors into the service's own errors. It
// returns nil for a nil error.
func memberError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrMemberNotFound
	case errors.Is(err, repository.ErrLastOwner):
		return ErrLastOwner
	case errors.Is(err, repository.ErrMembershipChanged):
		return ErrMembershipChanged
	default:
		return fmt.Errorf("changing membership: %w", err)
	}
}
