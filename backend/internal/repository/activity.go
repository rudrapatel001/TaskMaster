package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// ErrBadCursor is returned when a "before" cursor cannot be understood.
var ErrBadCursor = errors.New("invalid cursor")

// ActivityPage is one page of the feed, newest first.
type ActivityPage struct {
	Events     []models.ActivityEvent
	NextCursor *string // pass it back as "before" to get the next, older page; nil on the last page
}

type ActivityRepository struct {
	db *pgxpool.Pool
}

func NewActivityRepository(db *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// RecordTask writes a feed entry about a task. It looks the task up itself and
// stores a SNAPSHOT (identifier, title, and for moves the new column's name), so
// the entry still reads correctly after the task is renamed or deleted. Deleted
// tasks are found too, because "deleted" is recorded after the task is hidden.
func (r *ActivityRepository) RecordTask(ctx context.Context, e models.TaskEvent) error {
	var taskID, identifier, title, column string
	err := r.db.QueryRow(ctx, `
		SELECT t.id, p.key || '-' || t.number, t.title, s.name
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		JOIN project_statuses s ON s.id = t.status_id
		WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3`,
		e.ProjectID, e.WorkspaceID, e.Number).Scan(&taskID, &identifier, &title, &column)
	if err != nil {
		return fmt.Errorf("finding the task for the feed: %w", err)
	}

	metadata := map[string]any{"identifier": identifier, "title": title}
	if e.Action == "task.moved" {
		metadata["to"] = column
	}
	for key, value := range e.Extra {
		metadata[key] = value
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encoding the snapshot: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO activity_logs (workspace_id, actor_id, entity_type, entity_id, action, metadata)
		VALUES ($1, $2, 'task', $3, $4, $5::jsonb)`,
		e.WorkspaceID, e.ActorID, taskID, e.Action, string(encoded))
	if err != nil {
		return fmt.Errorf("inserting feed entry: %w", err)
	}
	return nil
}

// ListForWorkspace returns a page of everything that happened in the workspace.
func (r *ActivityRepository) ListForWorkspace(ctx context.Context, workspaceID string, limit int, cursor string) (*ActivityPage, error) {
	return r.list(ctx, workspaceID, "", limit, cursor)
}

// ListForTask returns a page of what happened to one task.
func (r *ActivityRepository) ListForTask(ctx context.Context, workspaceID, projectID string, number, limit int, cursor string) (*ActivityPage, error) {
	var taskID string
	err := r.db.QueryRow(ctx, `
		SELECT t.id
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3 AND t.deleted_at IS NULL`,
		projectID, workspaceID, number).Scan(&taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSuchTask
		}
		return nil, fmt.Errorf("finding task: %w", err)
	}
	return r.list(ctx, workspaceID, taskID, limit, cursor)
}

// list reads newest-first, using a cursor instead of an offset: "everything
// older than this entry". Unlike offsets, that stays correct while new entries
// keep arriving at the top.
func (r *ActivityRepository) list(ctx context.Context, workspaceID, taskID string, limit int, cursor string) (*ActivityPage, error) {
	conds := []string{"a.workspace_id = $1"}
	args := []any{workspaceID}

	if taskID != "" {
		args = append(args, taskID)
		conds = append(conds, fmt.Sprintf("a.entity_type = 'task' AND a.entity_id = $%d", len(args)))
	}
	if cursor != "" {
		at, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, ErrBadCursor
		}
		args = append(args, at, id)
		conds = append(conds, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", len(args)-1, len(args)))
	}

	// Ask for one more than needed: if it arrives, there is an older page.
	args = append(args, limit+1)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT a.id, a.action, a.entity_type, a.entity_id, a.actor_id,
		       COALESCE(u.name, 'Deleted user'), a.metadata, a.created_at
		FROM activity_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE %s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d`, strings.Join(conds, " AND "), len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("listing activity: %w", err)
	}
	defer rows.Close()

	events := []models.ActivityEvent{}
	for rows.Next() {
		var e models.ActivityEvent
		if err := rows.Scan(&e.ID, &e.Action, &e.EntityType, &e.EntityID, &e.ActorID,
			&e.ActorName, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("reading activity row: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading activity rows: %w", err)
	}

	page := &ActivityPage{Events: events}
	if len(events) > limit {
		events = events[:limit]
		page.Events = events
		last := events[len(events)-1]
		next := encodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	return page, nil
}

// A cursor is just "the time and id of the last entry you saw", made opaque.
func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || len(parts[1]) != 36 {
		return time.Time{}, "", errors.New("malformed cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return at, parts[1], nil
}
