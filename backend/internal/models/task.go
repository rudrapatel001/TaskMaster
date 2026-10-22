package models

import "time"

// Task is one piece of work in a project.
type Task struct {
	ID          string    `json:"id"`
	Identifier  string    `json:"identifier"` // the project key and number, like "WEB-12"
	Number      int       `json:"number"`     // 12: unique within the project, never reused
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StatusID    string    `json:"status_id"` // which board column it is in
	Priority    string    `json:"priority"`  // low, medium, high or urgent
	AssigneeID  *string   `json:"assignee_id"`
	ReporterID  *string   `json:"reporter_id"`
	DueDate     *string   `json:"due_date"` // "2026-10-20", or null
	Position    float64   `json:"position"` // order within its column; only meaningful next to its column-mates
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
