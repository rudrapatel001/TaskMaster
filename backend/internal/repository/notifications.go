package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// NotificationPage is one page of a person's notifications, newest first.
type NotificationPage struct {
	Notifications []models.Notification
	NextCursor    *string // nil on the last page
}

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// TaskPeople returns who a task concerns: its assignee and its reporter (either may be nil).
func (r *NotificationRepository) TaskPeople(ctx context.Context, workspaceID, projectID string, number int) (assignee, reporter *string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT t.assignee_id, t.reporter_id
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3 AND t.deleted_at IS NULL`,
		projectID, workspaceID, number).Scan(&assignee, &reporter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNoSuchTask
		}
		return nil, nil, fmt.Errorf("finding who the task concerns: %w", err)
	}
	return assignee, reporter, nil
}

// CreateForTask notifies each recipient about a task. The text is a SNAPSHOT
// (identifier, title, workspace slug for the link), like feed entries. People
// who are no longer in the workspace are silently skipped.
func (r *NotificationRepository) CreateForTask(ctx context.Context, e models.TaskEvent, kind string, recipients []string, extra map[string]any) error {
	var taskID, identifier, title, slug string
	err := r.db.QueryRow(ctx, `
		SELECT t.id, p.key || '-' || t.number, t.title, w.slug
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		JOIN workspaces w ON w.id = p.workspace_id
		WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3`,
		e.ProjectID, e.WorkspaceID, e.Number).Scan(&taskID, &identifier, &title, &slug)
	if err != nil {
		return fmt.Errorf("finding the task for the notification: %w", err)
	}

	metadata := map[string]any{"identifier": identifier, "title": title, "workspace_slug": slug}
	for key, value := range extra {
		metadata[key] = value
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encoding the snapshot: %w", err)
	}

	for _, recipient := range recipients {
		// The casts are needed because a bare SELECT of parameters has no types to infer from.
		_, err := r.db.Exec(ctx, `
			INSERT INTO notifications (user_id, workspace_id, actor_id, task_id, type, metadata)
			SELECT $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::text, $6::jsonb
			WHERE EXISTS (
				SELECT 1 FROM workspace_members WHERE workspace_id = $2::uuid AND user_id = $1::uuid
			)`,
			recipient, e.WorkspaceID, e.ActorID, taskID, kind, string(encoded))
		if err != nil {
			return fmt.Errorf("inserting notification: %w", err)
		}
	}
	return nil
}

// List returns a page of one person's notifications, newest first.
func (r *NotificationRepository) List(ctx context.Context, userID string, limit int, cursor string) (*NotificationPage, error) {
	conds := []string{"n.user_id = $1"}
	args := []any{userID}

	if cursor != "" {
		at, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, ErrBadCursor
		}
		args = append(args, at, id)
		conds = append(conds, fmt.Sprintf("(n.created_at, n.id) < ($%d, $%d)", len(args)-1, len(args)))
	}

	args = append(args, limit+1) // one extra row tells us whether there is an older page
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT n.id, n.type, n.actor_id, COALESCE(u.name, 'Deleted user'), n.metadata, n.read_at, n.created_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		WHERE %s
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $%d`, strings.Join(conds, " AND "), len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	defer rows.Close()

	list := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.ActorName, &n.Metadata, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("reading notification row: %w", err)
		}
		list = append(list, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading notification rows: %w", err)
	}

	page := &NotificationPage{Notifications: list}
	if len(list) > limit {
		list = list[:limit]
		page.Notifications = list
		last := list[len(list)-1]
		next := encodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	return page, nil
}

// UnreadCount is how many notifications a person has not read yet.
func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks one of the person's notifications as read. Doing it twice is harmless.
// The user id is part of the query, so nobody can touch someone else's notifications.
func (r *NotificationRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, now())
		WHERE id = $1 AND user_id = $2`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("marking notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks everything the person has as read.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	if _, err := r.db.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID); err != nil {
		return fmt.Errorf("marking all notifications read: %w", err)
	}
	return nil
}
