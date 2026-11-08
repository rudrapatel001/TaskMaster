package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// ErrNotificationNotFound means the person has no such notification.
var ErrNotificationNotFound = errors.New("notification not found")

// NotificationStore is what NotificationService needs from the data layer.
type NotificationStore interface {
	TaskPeople(ctx context.Context, workspaceID, projectID string, number int) (assignee, reporter *string, err error)
	CreateForTask(ctx context.Context, e models.TaskEvent, kind string, recipients []string, extra map[string]any) error
	List(ctx context.Context, userID string, limit int, cursor string) (*repository.NotificationPage, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type NotificationService struct {
	store  NotificationStore
	logger *slog.Logger
}

func NewNotificationService(store NotificationStore, logger *slog.Logger) *NotificationService {
	return &NotificationService{store: store, logger: logger}
}

// RecordTask looks at something that happened to a task and notifies the
// people it concerns:
//
//   - someone is given a task (when it is created, or its assignee is set)
//     -> the person who was given it
//   - a comment is posted -> the task's assignee and reporter
//
// Nobody is ever notified about their own action. Like the activity feed, this
// never returns an error: a failed notification must not break the action
// that caused it, so it is only logged.
func (s *NotificationService) RecordTask(ctx context.Context, e models.TaskEvent) {
	var kind string
	var recipients []string
	var extra map[string]any

	switch e.Action {
	case "task.created":
		if e.AssigneeID != nil {
			kind, recipients = "task.assigned", []string{*e.AssigneeID}
		}
	case "task.updated":
		// Only when the request itself set the assignee: the response always
		// names the current assignee, even if the edit was about something else.
		if e.AssigneeSent && e.AssigneeID != nil {
			kind, recipients = "task.assigned", []string{*e.AssigneeID}
		}
	case "task.commented":
		assignee, reporter, err := s.store.TaskPeople(ctx, e.WorkspaceID, e.ProjectID, e.Number)
		if err != nil {
			s.logger.Warn("could not find who a comment concerns", "error", err)
			return
		}
		kind, extra = "task.commented", e.Extra
		for _, person := range []*string{assignee, reporter} {
			if person != nil {
				recipients = append(recipients, *person)
			}
		}
	}

	recipients = everyoneButTheActor(recipients, e.ActorID)
	if kind == "" || len(recipients) == 0 {
		return
	}
	if err := s.store.CreateForTask(ctx, e, kind, recipients, extra); err != nil {
		s.logger.Warn("could not create notifications", "type", kind, "error", err)
	}
}

// everyoneButTheActor drops the person who did the thing, and duplicates (a
// person who is both assignee and reporter should be told once).
func everyoneButTheActor(ids []string, actorID string) []string {
	seen := map[string]bool{strings.ToLower(actorID): true}
	var others []string
	for _, id := range ids {
		key := strings.ToLower(id)
		if seen[key] {
			continue
		}
		seen[key] = true
		others = append(others, id)
	}
	return others
}

func (s *NotificationService) List(ctx context.Context, userID string, limit int, cursor string) (*repository.NotificationPage, error) {
	page, err := s.store.List(ctx, userID, limit, cursor)
	if err != nil {
		if errors.Is(err, repository.ErrBadCursor) {
			return nil, ErrInvalidCursor
		}
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	return page, nil
}

func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.store.UnreadCount(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID string) error {
	if err := s.store.MarkRead(ctx, userID, notificationID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotificationNotFound
		}
		return fmt.Errorf("marking notification read: %w", err)
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.store.MarkAllRead(ctx, userID)
}

// TaskEventSink receives task events. Both the activity feed and notifications do.
type TaskEventSink interface {
	RecordTask(ctx context.Context, e models.TaskEvent)
}

// TaskEvents hands every task event to several sinks, one after another. The
// router gives it to the recorder middleware, so one event feeds both.
type TaskEvents struct {
	sinks []TaskEventSink
}

func NewTaskEvents(sinks ...TaskEventSink) *TaskEvents {
	return &TaskEvents{sinks: sinks}
}

func (t *TaskEvents) RecordTask(ctx context.Context, e models.TaskEvent) {
	for _, sink := range t.sinks {
		sink.RecordTask(ctx, e)
	}
}
