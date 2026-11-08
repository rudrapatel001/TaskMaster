package models

import "time"

// Notification is a message for one person about something that concerns them.
type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"` // "task.assigned" or "task.commented"
	ActorID   *string        `json:"actor_id"`
	ActorName string         `json:"actor_name"`
	Metadata  map[string]any `json:"metadata"` // a snapshot: identifier, title, workspace_slug, ...
	ReadAt    *time.Time     `json:"read_at"`  // nil while unread
	CreatedAt time.Time      `json:"created_at"`
}
