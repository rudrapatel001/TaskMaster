package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

var (
	// ErrStatusNotFound means the project has no such column.
	ErrStatusNotFound = errors.New("column not found")

	// ErrStatusNameTaken means another column of the project already has that name.
	ErrStatusNameTaken = errors.New("column name already in use")

	// ErrTooManyStatuses means the project is at its limit of columns.
	ErrTooManyStatuses = errors.New("too many columns")

	// ErrLastStatus means the project's only column cannot be deleted.
	ErrLastStatus = errors.New("cannot delete the last column")

	// ErrColumnNotEmpty means the column has tasks and no place was given to move them to.
	ErrColumnNotEmpty = errors.New("column has tasks")

	// ErrInvalidTarget means the column to move tasks to is not another column of the project.
	ErrInvalidTarget = errors.New("invalid target column")

	// ErrInvalidStatusNeighbour means "before" is not another column of the project.
	ErrInvalidStatusNeighbour = errors.New("invalid neighbour column")
)

// StatusStore is what StatusService needs from the data layer.
type StatusStore interface {
	Create(ctx context.Context, workspaceID, projectID, name, category string) (*models.Status, error)
	Update(ctx context.Context, workspaceID, projectID, statusID string, name, category *string) (*models.Status, error)
	Move(ctx context.Context, workspaceID, projectID, statusID string, before *string) ([]models.Status, error)
	Delete(ctx context.Context, workspaceID, projectID, statusID string, moveTo *string) error
}

type StatusService struct {
	store StatusStore
}

func NewStatusService(store StatusStore) *StatusService {
	return &StatusService{store: store}
}

func (s *StatusService) Create(ctx context.Context, workspaceID, projectID, name, category string) (*models.Status, error) {
	st, err := s.store.Create(ctx, workspaceID, projectID, name, category)
	if err != nil {
		return nil, statusError(err)
	}
	return st, nil
}

func (s *StatusService) Update(ctx context.Context, workspaceID, projectID, statusID string, name, category *string) (*models.Status, error) {
	st, err := s.store.Update(ctx, workspaceID, projectID, statusID, name, category)
	if err != nil {
		return nil, statusError(err)
	}
	return st, nil
}

func (s *StatusService) Move(ctx context.Context, workspaceID, projectID, statusID string, before *string) ([]models.Status, error) {
	statuses, err := s.store.Move(ctx, workspaceID, projectID, statusID, before)
	if err != nil {
		return nil, statusError(err)
	}
	return statuses, nil
}

func (s *StatusService) Delete(ctx context.Context, workspaceID, projectID, statusID string, moveTo *string) error {
	return statusError(s.store.Delete(ctx, workspaceID, projectID, statusID, moveTo))
}

// statusError turns repository errors into the service's own errors. It
// returns nil for a nil error.
func statusError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNoSuchProject):
		return ErrProjectNotFound
	case errors.Is(err, repository.ErrNotFound):
		return ErrStatusNotFound
	case errors.Is(err, repository.ErrStatusNameTaken):
		return ErrStatusNameTaken
	case errors.Is(err, repository.ErrTooManyStatuses):
		return ErrTooManyStatuses
	case errors.Is(err, repository.ErrLastStatus):
		return ErrLastStatus
	case errors.Is(err, repository.ErrColumnNotEmpty):
		return ErrColumnNotEmpty
	case errors.Is(err, repository.ErrInvalidTarget):
		return ErrInvalidTarget
	case errors.Is(err, repository.ErrInvalidStatusNeighbour):
		return ErrInvalidStatusNeighbour
	default:
		return fmt.Errorf("column operation failed: %w", err)
	}
}
