package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

var (
	// ErrTaskNotFound means no such task exists in the project.
	ErrTaskNotFound = errors.New("task not found")

	// ErrInvalidStatus means the board column is not one of the project's columns.
	ErrInvalidStatus = errors.New("status does not belong to this project")

	// ErrAssigneeNotMember means the person is not in the workspace.
	ErrAssigneeNotMember = errors.New("assignee is not a member of the workspace")

	// ErrInvalidNeighbour means a move named a task that is not in the target column.
	ErrInvalidNeighbour = errors.New("neighbour is not a task in the target column")
)

// TaskStore is what TaskService needs from the data layer.
type TaskStore interface {
	Create(ctx context.Context, in repository.NewTask) (*models.Task, error)
	List(ctx context.Context, workspaceID, projectID string, f repository.TaskFilter) ([]models.Task, int, error)
	Get(ctx context.Context, workspaceID, projectID string, number int) (*models.Task, error)
	Update(ctx context.Context, workspaceID, projectID string, number int, p repository.TaskPatch) (*models.Task, error)
	Move(ctx context.Context, workspaceID, projectID string, number int, statusID string, before *int) (*models.Task, error)
	Delete(ctx context.Context, workspaceID, projectID string, number int) error
}

type TaskService struct {
	store TaskStore
}

func NewTaskService(store TaskStore) *TaskService {
	return &TaskService{store: store}
}

// Create adds a task. The input is already validated.
func (s *TaskService) Create(ctx context.Context, in repository.NewTask) (*models.Task, error) {
	t, err := s.store.Create(ctx, in)
	if err != nil {
		return nil, taskError(err)
	}
	return t, nil
}

func (s *TaskService) List(ctx context.Context, workspaceID, projectID string, f repository.TaskFilter) ([]models.Task, int, error) {
	tasks, total, err := s.store.List(ctx, workspaceID, projectID, f)
	if err != nil {
		return nil, 0, taskError(err)
	}
	return tasks, total, nil
}

func (s *TaskService) Get(ctx context.Context, workspaceID, projectID string, number int) (*models.Task, error) {
	t, err := s.store.Get(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, taskError(err)
	}
	return t, nil
}

func (s *TaskService) Update(ctx context.Context, workspaceID, projectID string, number int, p repository.TaskPatch) (*models.Task, error) {
	t, err := s.store.Update(ctx, workspaceID, projectID, number, p)
	if err != nil {
		return nil, taskError(err)
	}
	return t, nil
}

func (s *TaskService) Move(ctx context.Context, workspaceID, projectID string, number int, statusID string, before *int) (*models.Task, error) {
	t, err := s.store.Move(ctx, workspaceID, projectID, number, statusID, before)
	if err != nil {
		return nil, taskError(err)
	}
	return t, nil
}

func (s *TaskService) Delete(ctx context.Context, workspaceID, projectID string, number int) error {
	return taskError(s.store.Delete(ctx, workspaceID, projectID, number))
}

// taskError turns repository errors into the service's own errors. It returns
// nil for a nil error.
func taskError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNoSuchProject):
		return ErrProjectNotFound
	case errors.Is(err, repository.ErrNotFound):
		return ErrTaskNotFound
	case errors.Is(err, repository.ErrInvalidStatus):
		return ErrInvalidStatus
	case errors.Is(err, repository.ErrAssigneeNotMember):
		return ErrAssigneeNotMember
	case errors.Is(err, repository.ErrInvalidBefore):
		return ErrInvalidNeighbour
	default:
		return fmt.Errorf("task operation failed: %w", err)
	}
}
