package models

import "time"

// Invitation is an open invitation to join a workspace. It never contains the
// token: that is shown once, when the invitation is created.
type Invitation struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	InvitedBy string    `json:"invited_by"` // the inviter's name; empty if their account is gone
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InvitationPreview is what someone holding an invitation link may see before
// they log in.
type InvitationPreview struct {
	WorkspaceName string    `json:"workspace_name"`
	Email         string    `json:"email"`
	Role          Role      `json:"role"`
	InvitedBy     string    `json:"invited_by"`
	ExpiresAt     time.Time `json:"expires_at"`
}
