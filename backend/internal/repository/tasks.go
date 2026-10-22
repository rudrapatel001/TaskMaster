package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

var (
	// ErrNoSuchProject is returned when the project does not exist in the workspace.
	ErrNoSuchProject = errors.New("no such project")

	// ErrInvalidStatus is returned when a board column does not belong to the project.
	ErrInvalidStatus = errors.New("status does not belong to this project")

	// ErrAssigneeNotMember is returned when someone is assigned who is not in the workspace.
	ErrAssigneeNotMember = errors.New("assignee is not a member of the workspace")

	// ErrInvalidBefore is returned when a move names a neighbour that is not in the target column.
	ErrInvalidBefore = errors.New("neighbour is not a task in the target column")
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// taskSelect reads a task together with its project's key, which makes the
// identifier ("WEB-12").
const taskSelect = `
	SELECT t.id, p.key || '-' || t.number, t.number, t.title, t.description, t.status_id, t.priority,
	       t.assignee_id, t.reporter_id, to_char(t.due_date, 'YYYY-MM-DD'), t.position, t.created_at, t.updated_at
	FROM tasks t
	JOIN projects p ON p.id = t.project_id`

// scanner is satisfied by both a single row and a row in a result set.
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(row scanner) (*models.Task, error) {
	var t models.Task
	err := row.Scan(&t.ID, &t.Identifier, &t.Number, &t.Title, &t.Description, &t.StatusID, &t.Priority,
		&t.AssigneeID, &t.ReporterID, &t.DueDate, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// lockProject checks the project belongs to the workspace and locks it, so
// creating and moving tasks in one project happens one at a time. This is what
// keeps task numbers and board positions from colliding.
func lockProject(ctx context.Context, tx pgx.Tx, workspaceID, projectID string) error {
	var id string
	err := tx.QueryRow(ctx,
		`SELECT id FROM projects WHERE id = $1 AND workspace_id = $2 FOR NO KEY UPDATE`,
		projectID, workspaceID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoSuchProject
		}
		return fmt.Errorf("locking project: %w", err)
	}
	return nil
}

func statusBelongs(ctx context.Context, tx pgx.Tx, projectID, statusID string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM project_statuses WHERE id = $1 AND project_id = $2)`,
		statusID, projectID).Scan(&ok)
	return ok, err
}

func isWorkspaceMember(ctx context.Context, tx pgx.Tx, workspaceID, userID string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2)`,
		workspaceID, userID).Scan(&ok)
	return ok, err
}

// NewTask is everything needed to create a task.
type NewTask struct {
	WorkspaceID string
	ProjectID   string
	ReporterID  string
	Title       string
	Description string
	StatusID    string  // empty means the project's first column
	Priority    string  // already defaulted by the caller
	AssigneeID  *string // nil means unassigned
	DueDate     *string // "2026-10-20", or nil
}

// Create adds a task at the bottom of its column and gives it the next number
// in its project.
func (r *TaskRepository) Create(ctx context.Context, in NewTask) (*models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, in.WorkspaceID, in.ProjectID); err != nil {
		return nil, err
	}

	statusID := in.StatusID
	if statusID == "" {
		err := tx.QueryRow(ctx,
			`SELECT id FROM project_statuses WHERE project_id = $1 ORDER BY position, id LIMIT 1`,
			in.ProjectID).Scan(&statusID)
		if err != nil {
			return nil, fmt.Errorf("finding the first column: %w", err)
		}
	} else if ok, err := statusBelongs(ctx, tx, in.ProjectID, statusID); err != nil {
		return nil, fmt.Errorf("checking status: %w", err)
	} else if !ok {
		return nil, ErrInvalidStatus
	}

	if in.AssigneeID != nil {
		if ok, err := isWorkspaceMember(ctx, tx, in.WorkspaceID, *in.AssigneeID); err != nil {
			return nil, fmt.Errorf("checking assignee: %w", err)
		} else if !ok {
			return nil, ErrAssigneeNotMember
		}
	}

	// Take the next number. The project row is locked, so two people creating
	// tasks at the same moment get different numbers.
	var number int
	err = tx.QueryRow(ctx, `
		UPDATE projects SET next_task_number = next_task_number + 1
		WHERE id = $1
		RETURNING next_task_number - 1`, in.ProjectID).Scan(&number)
	if err != nil {
		return nil, fmt.Errorf("taking a task number: %w", err)
	}

	var position float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) + 1000
		FROM tasks
		WHERE project_id = $1 AND status_id = $2 AND deleted_at IS NULL`,
		in.ProjectID, statusID).Scan(&position)
	if err != nil {
		return nil, fmt.Errorf("finding the bottom of the column: %w", err)
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (project_id, number, title, description, status_id, priority,
		                   assignee_id, reporter_id, due_date, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::date, $10)
		RETURNING id`,
		in.ProjectID, number, in.Title, in.Description, statusID, in.Priority,
		in.AssigneeID, in.ReporterID, in.DueDate, position).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	task, err := scanTask(tx.QueryRow(ctx, taskSelect+` WHERE t.id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("reading the new task: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing task: %w", err)
	}
	return task, nil
}

// TaskFilter narrows and orders a task list. Empty fields mean "no filter".
type TaskFilter struct {
	StatusID   string
	AssigneeID string // a user id, or "none" for unassigned tasks
	Priority   string
	Query      string // matched against titles, ignoring capitals
	Sort       string // one of the keys of taskOrder
	Limit      int
	Offset     int
}

// taskOrder maps the allowed sort names to SQL. User input never reaches the
// query text: it only picks one of these.
var taskOrder = map[string]string{
	"position": "t.position, t.id",
	"newest":   "t.created_at DESC, t.id",
	"oldest":   "t.created_at, t.id",
	"due":      "t.due_date NULLS LAST, t.position, t.id",
}

// TaskSorts lists the valid sort names.
func TaskSorts() []string { return []string{"position", "newest", "oldest", "due"} }

// likeEscaper makes %, _ and \ in a search match themselves.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// List returns a page of a project's tasks, and how many match in total.
func (r *TaskRepository) List(ctx context.Context, workspaceID, projectID string, f TaskFilter) ([]models.Task, int, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1 AND workspace_id = $2)`,
		projectID, workspaceID).Scan(&exists)
	if err != nil {
		return nil, 0, fmt.Errorf("checking project: %w", err)
	}
	if !exists {
		return nil, 0, ErrNoSuchProject
	}

	conds := []string{"t.project_id = $1", "t.deleted_at IS NULL"}
	args := []any{projectID}
	add := func(condition string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(condition, len(args)))
	}
	if f.StatusID != "" {
		add("t.status_id = $%d", f.StatusID)
	}
	if f.AssigneeID == "none" {
		conds = append(conds, "t.assignee_id IS NULL")
	} else if f.AssigneeID != "" {
		add("t.assignee_id = $%d", f.AssigneeID)
	}
	if f.Priority != "" {
		add("t.priority = $%d", f.Priority)
	}
	if f.Query != "" {
		add("t.title ILIKE $%d", "%"+likeEscaper.Replace(f.Query)+"%")
	}
	where := strings.Join(conds, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM tasks t WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting tasks: %w", err)
	}

	order, ok := taskOrder[f.Sort]
	if !ok {
		order = taskOrder["position"]
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(taskSelect+` WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, where, order, len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("reading task row: %w", err)
		}
		tasks = append(tasks, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reading task rows: %w", err)
	}
	return tasks, total, nil
}

// Get returns one task by its number. The workspace is part of the query, so
// a task can only be reached through its own workspace.
func (r *TaskRepository) Get(ctx context.Context, workspaceID, projectID string, number int) (*models.Task, error) {
	t, err := scanTask(r.db.QueryRow(ctx,
		taskSelect+` WHERE p.id = $1 AND p.workspace_id = $2 AND t.number = $3 AND t.deleted_at IS NULL`,
		projectID, workspaceID, number))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying task: %w", err)
	}
	return t, nil
}

// TaskPatch describes a partial update. A nil pointer means "leave it alone".
// For the two nullable fields, Set says whether to change it at all, and a nil
// value then means "clear it".
type TaskPatch struct {
	Title       *string
	Description *string
	Priority    *string
	SetAssignee bool
	AssigneeID  *string
	SetDueDate  bool
	DueDate     *string
}

// Empty reports whether the patch changes nothing.
func (p TaskPatch) Empty() bool {
	return p.Title == nil && p.Description == nil && p.Priority == nil && !p.SetAssignee && !p.SetDueDate
}

// Update changes the fields named in the patch.
func (r *TaskRepository) Update(ctx context.Context, workspaceID, projectID string, number int, p TaskPatch) (*models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return nil, err
	}

	if p.SetAssignee && p.AssigneeID != nil {
		if ok, err := isWorkspaceMember(ctx, tx, workspaceID, *p.AssigneeID); err != nil {
			return nil, fmt.Errorf("checking assignee: %w", err)
		} else if !ok {
			return nil, ErrAssigneeNotMember
		}
	}

	// Build "SET col = $3, col = $4, ..." from the fields that were sent. Only
	// fixed column names go into the text; the values travel as parameters.
	args := []any{projectID, number}
	sets := []string{}
	set := func(column string, value any, cast string) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d%s", column, len(args), cast))
	}
	if p.Title != nil {
		set("title", *p.Title, "")
	}
	if p.Description != nil {
		set("description", *p.Description, "")
	}
	if p.Priority != nil {
		set("priority", *p.Priority, "")
	}
	if p.SetAssignee {
		set("assignee_id", p.AssigneeID, "")
	}
	if p.SetDueDate {
		set("due_date", p.DueDate, "::date")
	}
	sets = append(sets, "updated_at = now()")

	var id string
	err = tx.QueryRow(ctx,
		`UPDATE tasks SET `+strings.Join(sets, ", ")+`
		 WHERE project_id = $1 AND number = $2 AND deleted_at IS NULL
		 RETURNING id`, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating task: %w", err)
	}

	task, err := scanTask(tx.QueryRow(ctx, taskSelect+` WHERE t.id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("reading the updated task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing task update: %w", err)
	}
	return task, nil
}

// Move puts a task into a column, directly above the task numbered `before`
// there, or at the bottom if before is nil.
//
// Positions are fractional: dropping a card between two others gives it the
// number halfway between theirs, so only this one row changes. If repeated
// drops squeeze two positions too close together, the column is renumbered.
func (r *TaskRepository) Move(ctx context.Context, workspaceID, projectID string, number int, statusID string, before *int) (*models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProject(ctx, tx, workspaceID, projectID); err != nil {
		return nil, err
	}

	var taskID string
	err = tx.QueryRow(ctx,
		`SELECT id FROM tasks WHERE project_id = $1 AND number = $2 AND deleted_at IS NULL`,
		projectID, number).Scan(&taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("finding task: %w", err)
	}

	if ok, err := statusBelongs(ctx, tx, projectID, statusID); err != nil {
		return nil, fmt.Errorf("checking status: %w", err)
	} else if !ok {
		return nil, ErrInvalidStatus
	}

	var position float64
	if before == nil {
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(position), 0) + 1000
			FROM tasks
			WHERE project_id = $1 AND status_id = $2 AND deleted_at IS NULL AND id <> $3`,
			projectID, statusID, taskID).Scan(&position)
		if err != nil {
			return nil, fmt.Errorf("finding the bottom of the column: %w", err)
		}
	} else {
		// Two attempts: the second happens after a renumbering.
		for attempt := 0; attempt < 2; attempt++ {
			var upper float64
			err = tx.QueryRow(ctx, `
				SELECT position FROM tasks
				WHERE project_id = $1 AND status_id = $2 AND number = $3
				  AND deleted_at IS NULL AND id <> $4`,
				projectID, statusID, *before, taskID).Scan(&upper)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, ErrInvalidBefore
				}
				return nil, fmt.Errorf("finding the neighbour: %w", err)
			}

			var lower float64 // the position of whatever sits directly above, or 0 at the top
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(MAX(position), 0) FROM tasks
				WHERE project_id = $1 AND status_id = $2 AND deleted_at IS NULL
				  AND id <> $3 AND position < $4`,
				projectID, statusID, taskID, upper).Scan(&lower)
			if err != nil {
				return nil, fmt.Errorf("finding the task above: %w", err)
			}

			if upper-lower > 1e-6 {
				position = (lower + upper) / 2
				break
			}
			if attempt == 1 {
				return nil, fmt.Errorf("could not make room between positions %v and %v", lower, upper)
			}
			if err := renumberColumn(ctx, tx, projectID, statusID); err != nil {
				return nil, err
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tasks SET status_id = $2, position = $3, updated_at = now()
		WHERE id = $1`, taskID, statusID, position); err != nil {
		return nil, fmt.Errorf("moving task: %w", err)
	}

	task, err := scanTask(tx.QueryRow(ctx, taskSelect+` WHERE t.id = $1`, taskID))
	if err != nil {
		return nil, fmt.Errorf("reading the moved task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing move: %w", err)
	}
	return task, nil
}

// renumberColumn spreads a column's tasks back out to 1000, 2000, 3000, ...
// keeping their order.
func renumberColumn(ctx context.Context, tx pgx.Tx, projectID, statusID string) error {
	_, err := tx.Exec(ctx, `
		WITH ordered AS (
			SELECT id, row_number() OVER (ORDER BY position, id) AS n
			FROM tasks
			WHERE project_id = $1 AND status_id = $2 AND deleted_at IS NULL
		)
		UPDATE tasks t SET position = ordered.n * 1000
		FROM ordered WHERE t.id = ordered.id`,
		projectID, statusID)
	if err != nil {
		return fmt.Errorf("renumbering column: %w", err)
	}
	return nil
}

// Delete hides a task (a "soft" delete): the row stays, so it could be
// recovered, but it no longer appears anywhere. Its number is never reused.
func (r *TaskRepository) Delete(ctx context.Context, workspaceID, projectID string, number int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE tasks t SET deleted_at = now(), updated_at = now()
		FROM projects p
		WHERE p.id = t.project_id AND p.id = $1 AND p.workspace_id = $2
		  AND t.number = $3 AND t.deleted_at IS NULL`,
		projectID, workspaceID, number)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
