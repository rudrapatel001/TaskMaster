package models

import "time"

// Workspace is a team's shared space.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceMembership is a workspace together with the role the current user
// has in it.
type WorkspaceMembership struct {
	Workspace
	Role Role `json:"role"`
}
