package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

// ErrSlugTaken is returned when another workspace already uses the slug.
var ErrSlugTaken = errors.New("workspace slug already taken")

// Postgres error code for "a referenced row does not exist".
const pgForeignKeyViolation = "23503"

type WorkspaceRepository struct {
	db *pgxpool.Pool
}

func NewWorkspaceRepository(db *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create inserts a workspace and makes ownerID its owner, as one transaction.
func (r *WorkspaceRepository) Create(ctx context.Context, name, slug, ownerID string) (*models.WorkspaceMembership, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx) // does nothing once Commit has succeeded

	var w models.Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, created_at, updated_at`,
		name, slug, ownerID,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == "workspaces_slug_idx" {
				return nil, ErrSlugTaken
			}
			if pgErr.Code == pgForeignKeyViolation {
				return nil, ErrNotFound // the owner does not exist
			}
		}
		return nil, fmt.Errorf("inserting workspace: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)`,
		w.ID, ownerID, models.RoleOwner,
	)
	if err != nil {
		return nil, fmt.Errorf("adding owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing workspace: %w", err)
	}

	return &models.WorkspaceMembership{Workspace: w, Role: models.RoleOwner}, nil
}

// ListForUser returns every workspace the user belongs to, with their role in each.
func (r *WorkspaceRepository) ListForUser(ctx context.Context, userID string) ([]models.WorkspaceMembership, error) {
	rows, err := r.db.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.created_at, w.updated_at, m.role
		FROM workspace_members m
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id = $1
		ORDER BY w.created_at, w.id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}
	defer rows.Close()

	list := []models.WorkspaceMembership{} // never nil, so the JSON is [] and not null
	for rows.Next() {
		var m models.WorkspaceMembership
		if err := rows.Scan(&m.ID, &m.Name, &m.Slug, &m.CreatedAt, &m.UpdatedAt, &m.Role); err != nil {
			return nil, fmt.Errorf("reading workspace row: %w", err)
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading workspace rows: %w", err)
	}

	return list, nil
}
