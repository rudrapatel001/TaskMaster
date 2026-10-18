package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

var (
	// ErrProjectNotFound means no such project exists in this workspace.
	ErrProjectNotFound = errors.New("project not found")

	// ErrProjectKeyTaken means another project in the workspace already uses the key.
	ErrProjectKeyTaken = errors.New("project key already in use")
)

// ProjectStore is what ProjectService needs from the data layer.
type ProjectStore interface {
	Create(ctx context.Context, workspaceID, key, name, description, createdBy string) (*models.ProjectDetail, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]models.Project, error)
	Get(ctx context.Context, workspaceID, projectID string) (*models.ProjectDetail, error)
	Update(ctx context.Context, workspaceID, projectID, name, description string) (*models.Project, error)
	Delete(ctx context.Context, workspaceID, projectID string) error
}

type ProjectService struct {
	store ProjectStore
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{store: store}
}

// Create makes a project with its default columns. The input is already validated.
func (s *ProjectService) Create(ctx context.Context, workspaceID, userID, key, name, description string) (*models.ProjectDetail, error) {
	p, err := s.store.Create(ctx, workspaceID, key, name, description, userID)
	if err != nil {
		if errors.Is(err, repository.ErrProjectKeyTaken) {
			return nil, ErrProjectKeyTaken
		}
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, workspaceID string) ([]models.Project, error) {
	projects, err := s.store.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	return projects, nil
}

func (s *ProjectService) Get(ctx context.Context, workspaceID, projectID string) (*models.ProjectDetail, error) {
	p, err := s.store.Get(ctx, workspaceID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return p, nil
}

func (s *ProjectService) Update(ctx context.Context, workspaceID, projectID, name, description string) (*models.Project, error) {
	p, err := s.store.Update(ctx, workspaceID, projectID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("updating project: %w", err)
	}
	return p, nil
}

func (s *ProjectService) Delete(ctx context.Context, workspaceID, projectID string) error {
	if err := s.store.Delete(ctx, workspaceID, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("deleting project: %w", err)
	}
	return nil
}
