package models

import "time"

// ActivityEvent is one entry of the feed: something somebody did.
type ActivityEvent struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`      // like "task.moved"
	EntityType string         `json:"entity_type"` // what it happened to: "task"
	EntityID   string         `json:"entity_id"`
	ActorID    *string        `json:"actor_id"` // nil once the person's account is gone
	ActorName  string         `json:"actor_name"`
	Metadata   map[string]any `json:"metadata"` // a snapshot: identifier, title, ... as they were then
	CreatedAt  time.Time      `json:"created_at"`
}

// TaskEvent describes something that happened to a task, ready to be recorded.
type TaskEvent struct {
	WorkspaceID string
	ActorID     string
	ProjectID   string
	Number      int            // the task's number in its project
	Action      string         // "task.created", "task.moved", ...
	Extra       map[string]any // extra facts for the snapshot, such as a comment excerpt

	// What the recorder learned from the request and the response, when relevant.
	Fields       []string // task.updated: the fields the request tried to change
	AssigneeSent bool     // task.updated: the request included assignee_id
	AssigneeID   *string  // the task's assignee after the request (from the response)
}
