package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// ErrInvalidCursor means the "before" value of a feed request is not one we gave out.
var ErrInvalidCursor = errors.New("invalid cursor")

// ActivityStore is what ActivityService needs from the data layer.
type ActivityStore interface {
	RecordTask(ctx context.Context, e models.TaskEvent) error
	ListForWorkspace(ctx context.Context, workspaceID string, limit int, cursor string) (*repository.ActivityPage, error)
	ListForTask(ctx context.Context, workspaceID, projectID string, number, limit int, cursor string) (*repository.ActivityPage, error)
}

type ActivityService struct {
	store  ActivityStore
	logger *slog.Logger
}

func NewActivityService(store ActivityStore, logger *slog.Logger) *ActivityService {
	return &ActivityService{store: store, logger: logger}
}

// RecordTask adds an entry to the feed. It never returns an error: the feed is
// a convenience, so a failure to record must not break the action that caused
// it. It is logged instead.
func (s *ActivityService) RecordTask(ctx context.Context, e models.TaskEvent) {
	if err := s.store.RecordTask(ctx, e); err != nil {
		s.logger.Warn("could not record activity", "action", e.Action, "error", err)
	}
}

func (s *ActivityService) ListForWorkspace(ctx context.Context, workspaceID string, limit int, cursor string) (*repository.ActivityPage, error) {
	page, err := s.store.ListForWorkspace(ctx, workspaceID, limit, cursor)
	if err != nil {
		return nil, activityError(err)
	}
	return page, nil
}

func (s *ActivityService) ListForTask(ctx context.Context, workspaceID, projectID string, number, limit int, cursor string) (*repository.ActivityPage, error) {
	page, err := s.store.ListForTask(ctx, workspaceID, projectID, number, limit, cursor)
	if err != nil {
		return nil, activityError(err)
	}
	return page, nil
}

func activityError(err error) error {
	switch {
	case errors.Is(err, repository.ErrBadCursor):
		return ErrInvalidCursor
	case errors.Is(err, repository.ErrNoSuchTask):
		return ErrTaskNotFound
	default:
		return fmt.Errorf("activity operation failed: %w", err)
	}
}
