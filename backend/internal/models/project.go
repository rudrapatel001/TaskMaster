package models

import "time"

// Project is a body of work inside a workspace, such as "Website redesign".
type Project struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"` // short and permanent, like "WEB"; it prefixes task ids (WEB-12)
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Status is one column of a project's board, such as "In Progress".
type Status struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"` // todo, in_progress or done: what the column MEANS, whatever its name
	Position float64 `json:"position"` // columns are shown in ascending order
}

// ProjectDetail is a project together with its board columns.
type ProjectDetail struct {
	Project
	Statuses []Status `json:"statuses"`
}
