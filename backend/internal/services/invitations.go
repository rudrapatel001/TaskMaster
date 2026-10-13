package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rudrapatel001/taskmaster/backend/internal/auth"
	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/permissions"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

const (
	invitationTTL = 7 * 24 * time.Hour

	// A token is 32 random bytes in base64, which is always 43 characters.
	invitationTokenLength = 43
)

var (
	// ErrInvitationNotFound covers every way a link can be useless: unknown,
	// expired, revoked, or already used. They are deliberately the same.
	ErrInvitationNotFound = errors.New("invitation not found")

	// ErrAlreadyMember means the person is already in the workspace.
	ErrAlreadyMember = errors.New("already a member of this workspace")

	// ErrInvitationEmailMismatch means the logged-in account's email is not the
	// one the invitation was sent to.
	ErrInvitationEmailMismatch = errors.New("invitation is for a different email address")
)

// InvitationStore is what InvitationService needs from the data layer.
type InvitationStore interface {
	Create(ctx context.Context, workspaceID, email string, role models.Role, tokenHash, invitedBy string, expiresAt time.Time) (*models.Invitation, error)
	ListPending(ctx context.Context, workspaceID string) ([]models.Invitation, error)
	Revoke(ctx context.Context, workspaceID, invitationID string) error
	Preview(ctx context.Context, tokenHash string) (*models.InvitationPreview, error)
	Accept(ctx context.Context, tokenHash, userID string) (*models.WorkspaceMembership, error)
}

type InvitationService struct {
	store InvitationStore
}

func NewInvitationService(store InvitationStore) *InvitationService {
	return &InvitationService{store: store}
}

// Create invites an email address to the actor's workspace. It returns the
// invitation and the link token. The token exists only here: the database
// keeps just its hash, so it cannot be shown again.
func (s *InvitationService) Create(ctx context.Context, actor Actor, email string, role models.Role) (*models.Invitation, string, error) {
	if !permissions.CanInvite(actor.Role, role) {
		return nil, "", ErrForbidden
	}

	token, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, "", err
	}

	inv, err := s.store.Create(ctx, actor.WorkspaceID, email, role, hash, actor.UserID, time.Now().Add(invitationTTL))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAlreadyMember):
			return nil, "", ErrAlreadyMember
		case errors.Is(err, repository.ErrNotFound):
			return nil, "", ErrWorkspaceNotFound
		default:
			return nil, "", fmt.Errorf("creating invitation: %w", err)
		}
	}
	return inv, token, nil
}

// ListPending returns the workspace's open invitations.
func (s *InvitationService) ListPending(ctx context.Context, workspaceID string) ([]models.Invitation, error) {
	list, err := s.store.ListPending(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing invitations: %w", err)
	}
	return list, nil
}

// Revoke cancels an open invitation.
func (s *InvitationService) Revoke(ctx context.Context, workspaceID, invitationID string) error {
	if err := s.store.Revoke(ctx, workspaceID, invitationID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvitationNotFound
		}
		return fmt.Errorf("revoking invitation: %w", err)
	}
	return nil
}

// Preview shows what an invitation link is for, to anyone who holds it.
func (s *InvitationService) Preview(ctx context.Context, token string) (*models.InvitationPreview, error) {
	if len(token) != invitationTokenLength {
		return nil, ErrInvitationNotFound // cannot be a real token: skip the database
	}

	p, err := s.store.Preview(ctx, auth.HashOpaqueToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("previewing invitation: %w", err)
	}
	return p, nil
}

// Accept lets the logged-in user join the workspace the invitation is for.
func (s *InvitationService) Accept(ctx context.Context, token, userID string) (*models.WorkspaceMembership, error) {
	if len(token) != invitationTokenLength {
		return nil, ErrInvitationNotFound
	}

	m, err := s.store.Accept(ctx, auth.HashOpaqueToken(token), userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, ErrInvitationNotFound
		case errors.Is(err, repository.ErrEmailMismatch):
			return nil, ErrInvitationEmailMismatch
		case errors.Is(err, repository.ErrAlreadyMember):
			return nil, ErrAlreadyMember
		default:
			return nil, fmt.Errorf("accepting invitation: %w", err)
		}
	}
	return m, nil
}
