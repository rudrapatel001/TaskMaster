package models

import "time"

// Member is one person's membership in a workspace, as shown in a member list.
type Member struct {
	UserID   string    `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     Role      `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
