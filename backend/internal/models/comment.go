package models

import "time"

// Comment is one message in the discussion under a task.
type Comment struct {
	ID         string    `json:"id"`
	Body       string    `json:"body"`
	AuthorID   *string   `json:"author_id"`   // nil once the author's account is gone
	AuthorName string    `json:"author_name"` // "Deleted user" in that case
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}