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

// ErrSlugTaken is returned when another workspace already uses the slug.
var ErrSlugTaken = errors.New("workspace slug already taken")

var (
	// ErrLastOwner is returned when a change would leave a workspace without an owner.
	ErrLastOwner = errors.New("workspace must keep at least one owner")

	// ErrMembershipChanged is returned when someone's role is no longer what the
	// caller read a moment ago, because another request changed it first.
	ErrMembershipChanged = errors.New("membership changed")
)

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

// GetMembership returns a workspace together with the user's role in it.
// It returns ErrNotFound if the user is not a member, or the workspace does
// not exist: the two cases are deliberately indistinguishable.
func (r *WorkspaceRepository) GetMembership(ctx context.Context, workspaceID, userID string) (*models.WorkspaceMembership, error) {
	var m models.WorkspaceMembership
	err := r.db.QueryRow(ctx, `
		SELECT w.id, w.name, w.slug, w.created_at, w.updated_at, m.role
		FROM workspace_members m
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.workspace_id = $1 AND m.user_id = $2`,
		workspaceID, userID,
	).Scan(&m.ID, &m.Name, &m.Slug, &m.CreatedAt, &m.UpdatedAt, &m.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying membership: %w", err)
	}
	return &m, nil
}

// Rename changes a workspace's name. The slug stays the same, so links keep working.
func (r *WorkspaceRepository) Rename(ctx context.Context, workspaceID, name string) (*models.Workspace, error) {
	var w models.Workspace
	err := r.db.QueryRow(ctx, `
		UPDATE workspaces
		SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, created_at, updated_at`,
		workspaceID, name,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("renaming workspace: %w", err)
	}
	return &w, nil
}

// Delete removes a workspace. The database cascades the delete to everything
// inside it (members, projects, tasks, ...).
func (r *WorkspaceRepository) Delete(ctx context.Context, workspaceID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, workspaceID)
	if err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListMembers returns everyone in a workspace: owners first, then admins,
// members and viewers, each group in the order they joined.
func (r *WorkspaceRepository) ListMembers(ctx context.Context, workspaceID string) ([]models.Member, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.name, u.email, m.role, m.joined_at
		FROM workspace_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = $1
		ORDER BY
			CASE m.role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'member' THEN 3 ELSE 4 END,
			m.joined_at, u.id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	defer rows.Close()

	members := []models.Member{}
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("reading member row: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading member rows: %w", err)
	}
	return members, nil
}

// GetMember returns one member of a workspace.
func (r *WorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID string) (*models.Member, error) {
	var m models.Member
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.name, u.email, m.role, m.joined_at
		FROM workspace_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = $1 AND m.user_id = $2`,
		workspaceID, userID,
	).Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying member: %w", err)
	}
	return &m, nil
}

// SetMemberRole changes a member's role from expected to newRole.
//
// All membership changes in one workspace happen one at a time (the workspace
// row is locked), so the "at least one owner" check cannot be fooled by two
// simultaneous requests. If the member's role is no longer expected, nothing
// is changed and ErrMembershipChanged is returned.
func (r *WorkspaceRepository) SetMemberRole(ctx context.Context, workspaceID, userID string, expected, newRole models.Role) (*models.Member, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := lockAndReadRole(ctx, tx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if current != expected {
		return nil, ErrMembershipChanged
	}
	if current == models.RoleOwner && newRole != models.RoleOwner {
		if err := requireAnotherOwner(ctx, tx, workspaceID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE workspace_members SET role = $3
		WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID, newRole,
	); err != nil {
		return nil, fmt.Errorf("updating role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing role change: %w", err)
	}
	return r.GetMember(ctx, workspaceID, userID)
}

// RemoveMember removes a member whose role is expected. The same locking and
// "at least one owner" rules as SetMemberRole apply.
func (r *WorkspaceRepository) RemoveMember(ctx context.Context, workspaceID, userID string, expected models.Role) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := lockAndReadRole(ctx, tx, workspaceID, userID)
	if err != nil {
		return err
	}
	if current != expected {
		return ErrMembershipChanged
	}
	if current == models.RoleOwner {
		if err := requireAnotherOwner(ctx, tx, workspaceID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	); err != nil {
		return fmt.Errorf("removing member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing removal: %w", err)
	}
	return nil
}

// lockAndReadRole locks the workspace row, so membership changes in this
// workspace queue up one behind another, and returns the member's current role.
func lockAndReadRole(ctx context.Context, tx pgx.Tx, workspaceID, userID string) (models.Role, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM workspaces WHERE id = $1 FOR NO KEY UPDATE`, workspaceID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("locking workspace: %w", err)
	}

	var role models.Role
	err = tx.QueryRow(ctx, `
		SELECT role FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("reading member role: %w", err)
	}
	return role, nil
}

// requireAnotherOwner returns ErrLastOwner unless the workspace has more than
// one owner. Call it before demoting or removing an owner.
func requireAnotherOwner(ctx context.Context, tx pgx.Tx, workspaceID string) error {
	var owners int
	err := tx.QueryRow(ctx, `
		SELECT count(*) FROM workspace_members
		WHERE workspace_id = $1 AND role = 'owner'`,
		workspaceID,
	).Scan(&owners)
	if err != nil {
		return fmt.Errorf("counting owners: %w", err)
	}
	if owners <= 1 {
		return ErrLastOwner
	}
	return nil
}
