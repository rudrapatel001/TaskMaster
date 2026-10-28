package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// ErrNoSuchTask is returned when the task does not exist in the project (or was deleted).
var ErrNoSuchTask = errors.New("no such task")

type CommentRepository struct {
	db *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

// commentSelect reads a comment with its author's name. The author may be gone
// (the column is emptied when an account is deleted), hence the LEFT JOIN.
const commentSelect = `
	SELECT c.id, c.body, c.author_id, COALESCE(u.name, 'Deleted user'), c.created_at, c.updated_at
	FROM comments c
	LEFT JOIN users u ON u.id = c.author_id`

func scanComment(row scanner) (*models.Comment, error) {
	var c models.Comment
	if err := row.Scan(&c.ID, &c.Body, &c.AuthorID, &c.AuthorName, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// taskID finds a task's id from its number. It goes through the project and the
// workspace on purpose: comments can only be reached via their own workspace.
func (r *CommentRepository) taskID(ctx context.Context, workspaceID, projectID string, number int) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT t.id
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3 AND t.deleted_at IS NULL`,
		projectID, workspaceID, number).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNoSuchTask
		}
		return "", fmt.Errorf("finding task: %w", err)
	}
	return id, nil
}

// get reads one comment of a task that has already been located.
func (r *CommentRepository) get(ctx context.Context, taskID, commentID string) (*models.Comment, error) {
	c, err := scanComment(r.db.QueryRow(ctx, commentSelect+` WHERE c.id = $1 AND c.task_id = $2`, commentID, taskID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying comment: %w", err)
	}
	return c, nil
}

// List returns a task's comments, oldest first.
func (r *CommentRepository) List(ctx context.Context, workspaceID, projectID string, number int) ([]models.Comment, error) {
	taskID, err := r.taskID(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, commentSelect+` WHERE c.task_id = $1 ORDER BY c.created_at, c.id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	defer rows.Close()

	comments := []models.Comment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("reading comment row: %w", err)
		}
		comments = append(comments, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading comment rows: %w", err)
	}
	return comments, nil
}

// Create adds a comment to a task.
func (r *CommentRepository) Create(ctx context.Context, workspaceID, projectID string, number int, authorID, body string) (*models.Comment, error) {
	taskID, err := r.taskID(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, err
	}

	var id string
	err = r.db.QueryRow(ctx, `
		INSERT INTO comments (task_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id`,
		taskID, authorID, body).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("inserting comment: %w", err)
	}
	return r.get(ctx, taskID, id)
}

// Get returns one comment of a task.
func (r *CommentRepository) Get(ctx context.Context, workspaceID, projectID string, number int, commentID string) (*models.Comment, error) {
	taskID, err := r.taskID(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, err
	}
	return r.get(ctx, taskID, commentID)
}

// Update changes a comment's text.
func (r *CommentRepository) Update(ctx context.Context, workspaceID, projectID string, number int, commentID, body string) (*models.Comment, error) {
	taskID, err := r.taskID(ctx, workspaceID, projectID, number)
	if err != nil {
		return nil, err
	}

	var id string
	err = r.db.QueryRow(ctx, `
		UPDATE comments SET body = $3, updated_at = now()
		WHERE id = $1 AND task_id = $2
		RETURNING id`,
		commentID, taskID, body).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating comment: %w", err)
	}
	return r.get(ctx, taskID, id)
}

// Delete removes a comment for good.
func (r *CommentRepository) Delete(ctx context.Context, workspaceID, projectID string, number int, commentID string) error {
	taskID, err := r.taskID(ctx, workspaceID, projectID, number)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx, `DELETE FROM comments WHERE id = $1 AND task_id = $2`, commentID, taskID)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
