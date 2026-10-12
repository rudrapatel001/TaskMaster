package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

const maxSlugAttempts = 5

var (
	// ErrSlugUnavailable is returned if no free slug could be found. In
	// practice this does not happen.
	ErrSlugUnavailable = errors.New("could not find a free workspace slug")

	// ErrWorkspaceNotFound means the workspace does not exist, or the person
	// asking is not a member of it. The two are deliberately the same.
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

// WorkspaceStore is what WorkspaceService needs from the data layer.
type WorkspaceStore interface {
	Create(ctx context.Context, name, slug, ownerID string) (*models.WorkspaceMembership, error)
	ListForUser(ctx context.Context, userID string) ([]models.WorkspaceMembership, error)
	GetMembership(ctx context.Context, workspaceID, userID string) (*models.WorkspaceMembership, error)
	Rename(ctx context.Context, workspaceID, name string) (*models.Workspace, error)
	Delete(ctx context.Context, workspaceID string) error
	ListMembers(ctx context.Context, workspaceID string) ([]models.Member, error)
}

type WorkspaceService struct {
	store WorkspaceStore
}

func NewWorkspaceService(store WorkspaceStore) *WorkspaceService {
	return &WorkspaceService{store: store}
}

// Create makes a new workspace and its owner. name must already be validated.
func (s *WorkspaceService) Create(ctx context.Context, ownerID, name string) (*models.WorkspaceMembership, error) {
	base := slugify(name)

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 0 {
			suffix, err := randomSuffix()
			if err != nil {
				return nil, err
			}
			slug = base + "-" + suffix
		}

		created, err := s.store.Create(ctx, name, slug, ownerID)
		switch {
		case err == nil:
			return created, nil
		case errors.Is(err, repository.ErrSlugTaken):
			continue // someone has this slug: try again with a random suffix
		case errors.Is(err, repository.ErrNotFound):
			return nil, ErrUserNotFound
		default:
			return nil, fmt.Errorf("creating workspace: %w", err)
		}
	}

	return nil, ErrSlugUnavailable
}

// ListForUser returns the workspaces the user belongs to.
func (s *WorkspaceService) ListForUser(ctx context.Context, userID string) ([]models.WorkspaceMembership, error) {
	list, err := s.store.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}
	return list, nil
}

// GetMembership returns the workspace and the user's role in it, or
// ErrWorkspaceNotFound if the user is not a member.
func (s *WorkspaceService) GetMembership(ctx context.Context, workspaceID, userID string) (*models.WorkspaceMembership, error) {
	m, err := s.store.GetMembership(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("getting membership: %w", err)
	}
	return m, nil
}

// Rename changes the workspace's name. The caller has already been authorized.
func (s *WorkspaceService) Rename(ctx context.Context, workspaceID, name string) (*models.Workspace, error) {
	w, err := s.store.Rename(ctx, workspaceID, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("renaming workspace: %w", err)
	}
	return w, nil
}

// Delete removes the workspace and everything in it. The caller has already
// been authorized.
func (s *WorkspaceService) Delete(ctx context.Context, workspaceID string) error {
	if err := s.store.Delete(ctx, workspaceID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrWorkspaceNotFound
		}
		return fmt.Errorf("deleting workspace: %w", err)
	}
	return nil
}

// ListMembers returns everyone in the workspace.
func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID string) ([]models.Member, error) {
	members, err := s.store.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	return members, nil
}

// randomSuffix returns 4 random hex characters, like "9f3a".
func randomSuffix() (string, error) {
	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating slug suffix: %w", err)
	}
	return hex.EncodeToString(b), nil
}
