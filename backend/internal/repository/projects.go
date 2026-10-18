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

// ErrProjectKeyTaken is returned when the workspace already has a project with that key.
var ErrProjectKeyTaken = errors.New("project key already taken")

// defaultStatuses are the columns every new project starts with. The positions
// are spaced 1000 apart so columns can be inserted between them later.
var defaultStatuses = []struct {
	name     string
	category string
	position float64
}{
	{"To Do", "todo", 1000},
	{"In Progress", "in_progress", 2000},
	{"Review", "in_progress", 3000},
	{"Done", "done", 4000},
}

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a project and its default columns, as one transaction.
func (r *ProjectRepository) Create(ctx context.Context, workspaceID, key, name, description, createdBy string) (*models.ProjectDetail, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var p models.Project
	err = tx.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, description, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, key, name, description, created_at, updated_at`,
		workspaceID, key, name, description, createdBy,
	).Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation &&
			pgErr.ConstraintName == "projects_workspace_key_unique" {
			return nil, ErrProjectKeyTaken
		}
		return nil, fmt.Errorf("inserting project: %w", err)
	}

	statuses := make([]models.Status, 0, len(defaultStatuses))
	for _, d := range defaultStatuses {
		var s models.Status
		err := tx.QueryRow(ctx, `
			INSERT INTO project_statuses (project_id, name, category, position)
			VALUES ($1, $2, $3, $4)
			RETURNING id, name, category, position`,
			p.ID, d.name, d.category, d.position,
		).Scan(&s.ID, &s.Name, &s.Category, &s.Position)
		if err != nil {
			return nil, fmt.Errorf("inserting default status %q: %w", d.name, err)
		}
		statuses = append(statuses, s)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing project: %w", err)
	}
	return &models.ProjectDetail{Project: p, Statuses: statuses}, nil
}

// ListByWorkspace returns the workspace's projects, oldest first.
func (r *ProjectRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]models.Project, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, description, created_at, updated_at
		FROM projects
		WHERE workspace_id = $1
		ORDER BY created_at, id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()

	projects := []models.Project{}
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("reading project row: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading project rows: %w", err)
	}
	return projects, nil
}

// Get returns one project with its columns. The workspace id is part of the
// query on purpose: a project can only be reached through its own workspace.
func (r *ProjectRepository) Get(ctx context.Context, workspaceID, projectID string) (*models.ProjectDetail, error) {
	var p models.Project
	err := r.db.QueryRow(ctx, `
		SELECT id, key, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1 AND workspace_id = $2`,
		projectID, workspaceID,
	).Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying project: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, category, position
		FROM project_statuses
		WHERE project_id = $1
		ORDER BY position, id`,
		p.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing statuses: %w", err)
	}
	defer rows.Close()

	statuses := []models.Status{}
	for rows.Next() {
		var s models.Status
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Position); err != nil {
			return nil, fmt.Errorf("reading status row: %w", err)
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading status rows: %w", err)
	}

	return &models.ProjectDetail{Project: p, Statuses: statuses}, nil
}

// Update changes a project's name and description. The key never changes.
func (r *ProjectRepository) Update(ctx context.Context, workspaceID, projectID, name, description string) (*models.Project, error) {
	var p models.Project
	err := r.db.QueryRow(ctx, `
		UPDATE projects
		SET name = $3, description = $4, updated_at = now()
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, key, name, description, created_at, updated_at`,
		projectID, workspaceID, name, description,
	).Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating project: %w", err)
	}
	return &p, nil
}

// Delete removes a project. The database cascades to its columns and tasks.
func (r *ProjectRepository) Delete(ctx context.Context, workspaceID, projectID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1 AND workspace_id = $2`, projectID, workspaceID)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
