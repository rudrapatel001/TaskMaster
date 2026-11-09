package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// MaxStatuses is how many board columns a project may have.
const MaxStatuses = 12

var (
	// ErrStatusNameTaken is returned when the project already has a column with that name.
	ErrStatusNameTaken = errors.New("a column with this name already exists")

	// ErrTooManyStatuses is returned when a project is already at MaxStatuses.
	ErrTooManyStatuses = errors.New("project already has the maximum number of columns")

	// ErrLastStatus is returned when someone tries to delete a project's only column.
	ErrLastStatus = errors.New("a project must keep at least one column")

	// ErrColumnNotEmpty is returned when a column with tasks is deleted without saying where they go.
	ErrColumnNotEmpty = errors.New("column still has tasks")

	// ErrInvalidTarget is returned when the column to move tasks to is not another column of the project.
	ErrInvalidTarget = errors.New("target is not another column of this project")

	// ErrInvalidStatusNeighbour is returned when "before" is not another column of the project.
	ErrInvalidStatusNeighbour = errors.New("neighbour is not another column of this project")
)

type StatusRepository struct {
	db *pgxpool.Pool
}

func NewStatusRepository(db *pgxpool.Pool) *StatusRepository {
	return &StatusRepository{db: db}
}

// isUniqueViolation reports whether err is "that name is already taken".
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

// Create adds a column at the end of the board.
func (r *StatusRepository) Create(ctx context.Context, workspaceID, projectID, name, category string) (*models.Status, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return nil, err
	}

	var count int
	var last float64
	err = tx.QueryRow(ctx, `
		SELECT count(*), COALESCE(MAX(position), 0)
		FROM project_statuses WHERE project_id = $1`, projectID).Scan(&count, &last)
	if err != nil {
		return nil, fmt.Errorf("counting columns: %w", err)
	}
	if count >= MaxStatuses {
		return nil, ErrTooManyStatuses
	}

	var s models.Status
	err = tx.QueryRow(ctx, `
		INSERT INTO project_statuses (project_id, name, category, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, category, position`,
		projectID, name, category, last+1000).Scan(&s.ID, &s.Name, &s.Category, &s.Position)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrStatusNameTaken
		}
		return nil, fmt.Errorf("inserting column: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing column: %w", err)
	}
	return &s, nil
}

// Update renames a column and/or changes its category. A nil argument means
// "leave that as it is".
func (r *StatusRepository) Update(ctx context.Context, workspaceID, projectID, statusID string, name, category *string) (*models.Status, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return nil, err
	}

	var s models.Status
	err = tx.QueryRow(ctx, `
		UPDATE project_statuses
		SET name = COALESCE($3::text, name), category = COALESCE($4::text, category)
		WHERE id = $1 AND project_id = $2
		RETURNING id, name, category, position`,
		statusID, projectID, name, category).Scan(&s.ID, &s.Name, &s.Category, &s.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrStatusNameTaken
		}
		return nil, fmt.Errorf("updating column: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing column update: %w", err)
	}
	return &s, nil
}

// reorder returns the ids with "moved" taken out and put directly before
// "before", or last if before is nil. It is a pure function, so it is easy to test.
func reorder(ids []string, moved string, before *string) ([]string, error) {
	rest := make([]string, 0, len(ids))
	found := false
	for _, id := range ids {
		if id == moved {
			found = true
			continue
		}
		rest = append(rest, id)
	}
	if !found {
		return nil, ErrNotFound
	}

	if before == nil {
		return append(rest, moved), nil
	}
	for i, id := range rest {
		if id == *before {
			result := make([]string, 0, len(ids))
			result = append(result, rest[:i]...)
			result = append(result, moved)
			result = append(result, rest[i:]...)
			return result, nil
		}
	}
	return nil, ErrInvalidStatusNeighbour
}

// listStatuses reads a project's columns in board order.
func listStatuses(ctx context.Context, tx pgx.Tx, projectID string) ([]models.Status, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, name, category, position
		FROM project_statuses
		WHERE project_id = $1
		ORDER BY position, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	statuses := []models.Status{}
	for rows.Next() {
		var s models.Status
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Position); err != nil {
			return nil, fmt.Errorf("reading column row: %w", err)
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading column rows: %w", err)
	}
	return statuses, nil
}

// Move puts a column directly before another column, or last if before is nil,
// and returns all the columns in their new order. A project has few columns,
// so every column is simply renumbered 1000, 2000, 3000, ... in the new order.
func (r *StatusRepository) Move(ctx context.Context, workspaceID, projectID, statusID string, before *string) ([]models.Status, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return nil, err
	}

	statuses, err := listStatuses(ctx, tx, projectID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.Status, len(statuses))
	ids := make([]string, 0, len(statuses))
	for _, s := range statuses {
		byID[s.ID] = s
		ids = append(ids, s.ID)
	}

	order, err := reorder(ids, statusID, before)
	if err != nil {
		return nil, err
	}

	result := make([]models.Status, 0, len(order))
	for i, id := range order {
		position := float64((i + 1) * 1000)
		if _, err := tx.Exec(ctx,
			`UPDATE project_statuses SET position = $1 WHERE id = $2 AND project_id = $3`,
			position, id, projectID); err != nil {
			return nil, fmt.Errorf("renumbering columns: %w", err)
		}
		s := byID[id]
		s.Position = position
		result = append(result, s)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing column order: %w", err)
	}
	return result, nil
}

// Delete removes a column. Tasks in it move to the bottom of moveTo, keeping
// their order. That includes tasks that were deleted (hidden) earlier: the
// database refuses to delete a column anything still points to.
//
// If the column has live tasks, moveTo is required. If it has none, moveTo may
// be nil, and any hidden tasks go to the project's first other column.
func (r *StatusRepository) Delete(ctx context.Context, workspaceID, projectID, statusID string, moveTo *string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return err
	}

	statuses, err := listStatuses(ctx, tx, projectID)
	if err != nil {
		return err
	}
	exists := false
	firstOther := ""
	targetOK := false
	for _, s := range statuses {
		if s.ID == statusID {
			exists = true
			continue
		}
		if firstOther == "" {
			firstOther = s.ID
		}
		if moveTo != nil && s.ID == *moveTo {
			targetOK = true
		}
	}
	if !exists {
		return ErrNotFound
	}
	if len(statuses) < 2 {
		return ErrLastStatus
	}

	var live, total int
	err = tx.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE deleted_at IS NULL), count(*)
		FROM tasks WHERE status_id = $1`, statusID).Scan(&live, &total)
	if err != nil {
		return fmt.Errorf("counting tasks in the column: %w", err)
	}

	target := ""
	switch {
	case moveTo != nil:
		if !targetOK {
			return ErrInvalidTarget
		}
		target = *moveTo
	case live > 0:
		return ErrColumnNotEmpty
	case total > 0:
		target = firstOther // only hidden tasks: they must still go somewhere
	}

	if total > 0 {
		var base float64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(position), 0) FROM tasks
			WHERE project_id = $1 AND status_id = $2`, projectID, target).Scan(&base)
		if err != nil {
			return fmt.Errorf("finding the bottom of the target column: %w", err)
		}

		// Everything from the doomed column goes below what is already there, in its old order.
		if _, err := tx.Exec(ctx, `
			WITH moving AS (
				SELECT id, row_number() OVER (ORDER BY position, id) AS n
				FROM tasks WHERE status_id = $1
			)
			UPDATE tasks t
			SET status_id = $2, position = $3::float8 + moving.n * 1000, updated_at = now()
			FROM moving WHERE t.id = moving.id`, statusID, target, base); err != nil {
			return fmt.Errorf("moving tasks out of the column: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM project_statuses WHERE id = $1 AND project_id = $2`, statusID, projectID); err != nil {
		return fmt.Errorf("deleting column: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing column deletion: %w", err)
	}
	return nil
}
