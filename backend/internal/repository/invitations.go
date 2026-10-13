package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
)

var (
	// ErrAlreadyMember is returned when the person is already in the workspace.
	ErrAlreadyMember = errors.New("already a member")

	// ErrEmailMismatch is returned when an invitation was sent to a different
	// email address than the account trying to accept it.
	ErrEmailMismatch = errors.New("invitation is for a different email address")
)

type InvitationRepository struct {
	db *pgxpool.Pool
}

func NewInvitationRepository(db *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create stores a new invitation. Any open invitation for the same email in the
// same workspace is revoked first, so there is only ever one live link.
func (r *InvitationRepository) Create(ctx context.Context, workspaceID, email string, role models.Role, tokenHash, invitedBy string, expiresAt time.Time) (*models.Invitation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Queue behind other membership changes in this workspace.
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM workspaces WHERE id = $1 FOR NO KEY UPDATE`, workspaceID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("locking workspace: %w", err)
	}

	var alreadyMember bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspace_members m
			JOIN users u ON u.id = m.user_id
			WHERE m.workspace_id = $1 AND lower(u.email) = lower($2)
		)`, workspaceID, email,
	).Scan(&alreadyMember)
	if err != nil {
		return nil, fmt.Errorf("checking membership: %w", err)
	}
	if alreadyMember {
		return nil, ErrAlreadyMember
	}

	if _, err := tx.Exec(ctx, `
		UPDATE invitations SET revoked_at = now()
		WHERE workspace_id = $1 AND lower(email) = lower($2)
		  AND accepted_at IS NULL AND revoked_at IS NULL`,
		workspaceID, email,
	); err != nil {
		return nil, fmt.Errorf("revoking older invitation: %w", err)
	}

	var inv models.Invitation
	err = tx.QueryRow(ctx, `
		INSERT INTO invitations (workspace_id, email, role, token_hash, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, role, created_at, expires_at,
		          COALESCE((SELECT name FROM users WHERE id = $5), '')`,
		workspaceID, email, role, tokenHash, invitedBy, expiresAt,
	).Scan(&inv.ID, &inv.Email, &inv.Role, &inv.CreatedAt, &inv.ExpiresAt, &inv.InvitedBy)
	if err != nil {
		return nil, fmt.Errorf("inserting invitation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing invitation: %w", err)
	}
	return &inv, nil
}

// ListPending returns the open, unexpired invitations of a workspace, newest first.
func (r *InvitationRepository) ListPending(ctx context.Context, workspaceID string) ([]models.Invitation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.email, i.role, COALESCE(u.name, ''), i.created_at, i.expires_at
		FROM invitations i
		LEFT JOIN users u ON u.id = i.invited_by
		WHERE i.workspace_id = $1
		  AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now()
		ORDER BY i.created_at DESC, i.id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing invitations: %w", err)
	}
	defer rows.Close()

	list := []models.Invitation{}
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
			return nil, fmt.Errorf("reading invitation row: %w", err)
		}
		list = append(list, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading invitation rows: %w", err)
	}
	return list, nil
}

// Revoke cancels an open invitation of the given workspace.
func (r *InvitationRepository) Revoke(ctx context.Context, workspaceID, invitationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE invitations SET revoked_at = now()
		WHERE id = $1 AND workspace_id = $2
		  AND accepted_at IS NULL AND revoked_at IS NULL`,
		invitationID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("revoking invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Preview returns what an invitation link shows before login. It returns
// ErrNotFound unless the invitation is open and unexpired.
func (r *InvitationRepository) Preview(ctx context.Context, tokenHash string) (*models.InvitationPreview, error) {
	var p models.InvitationPreview
	err := r.db.QueryRow(ctx, `
		SELECT w.name, i.email, i.role, COALESCE(u.name, ''), i.expires_at
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		LEFT JOIN users u ON u.id = i.invited_by
		WHERE i.token_hash = $1
		  AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now()`,
		tokenHash,
	).Scan(&p.WorkspaceName, &p.Email, &p.Role, &p.InvitedBy, &p.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying invitation: %w", err)
	}
	return &p, nil
}

// Accept uses an invitation: it adds the user to the workspace with the
// invited role and marks the invitation as used, as one transaction. The
// invitation row is locked meanwhile, so it can only be used once even if
// several requests arrive at the same moment.
func (r *InvitationRepository) Accept(ctx context.Context, tokenHash, userID string) (*models.WorkspaceMembership, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var invitationID, workspaceID, invitedEmail string
	var role models.Role
	err = tx.QueryRow(ctx, `
		SELECT id, workspace_id, email, role
		FROM invitations
		WHERE token_hash = $1
		  AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()
		FOR UPDATE`,
		tokenHash,
	).Scan(&invitationID, &workspaceID, &invitedEmail, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("locking invitation: %w", err)
	}

	var userEmail string
	err = tx.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&userEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading user: %w", err)
	}
	if !strings.EqualFold(userEmail, invitedEmail) {
		return nil, ErrEmailMismatch
	}

	var alreadyMember bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2)`,
		workspaceID, userID,
	).Scan(&alreadyMember)
	if err != nil {
		return nil, fmt.Errorf("checking membership: %w", err)
	}
	if alreadyMember {
		return nil, ErrAlreadyMember
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)`,
		workspaceID, userID, role,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, ErrAlreadyMember
		}
		return nil, fmt.Errorf("adding member: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE invitations SET accepted_at = now() WHERE id = $1`, invitationID); err != nil {
		return nil, fmt.Errorf("marking invitation accepted: %w", err)
	}

	var m models.WorkspaceMembership
	err = tx.QueryRow(ctx, `
		SELECT w.id, w.name, w.slug, w.created_at, w.updated_at, m.role
		FROM workspace_members m
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.workspace_id = $1 AND m.user_id = $2`,
		workspaceID, userID,
	).Scan(&m.ID, &m.Name, &m.Slug, &m.CreatedAt, &m.UpdatedAt, &m.Role)
	if err != nil {
		return nil, fmt.Errorf("reading new membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing acceptance: %w", err)
	}
	return &m, nil
}
