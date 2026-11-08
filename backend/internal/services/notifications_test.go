package services

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rudrapatel001/taskmaster/backend/internal/models"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
)

// fakeNotificationStore records what the service asked it to create.
type fakeNotificationStore struct {
	assignee, reporter *string

	kind       string
	recipients []string
	extra      map[string]any
	created    int
}

func (f *fakeNotificationStore) TaskPeople(context.Context, string, string, int) (*string, *string, error) {
	return f.assignee, f.reporter, nil
}

func (f *fakeNotificationStore) CreateForTask(_ context.Context, _ models.TaskEvent, kind string, recipients []string, extra map[string]any) error {
	f.kind, f.recipients, f.extra = kind, recipients, extra
	f.created++
	return nil
}

func (f *fakeNotificationStore) List(context.Context, string, int, string) (*repository.NotificationPage, error) {
	return nil, nil
}
func (f *fakeNotificationStore) UnreadCount(context.Context, string) (int, error) { return 0, nil }
func (f *fakeNotificationStore) MarkRead(context.Context, string, string) error   { return nil }
func (f *fakeNotificationStore) MarkAllRead(context.Context, string) error        { return nil }

func ptr(s string) *string { return &s }

func TestWhoGetsNotified(t *testing.T) {
	const actor, asha, mihir = "actor-1", "asha-2", "mihir-3"

	cases := []struct {
		name           string
		event          models.TaskEvent
		assignee       *string // who the task is assigned to, for comment events
		reporter       *string
		wantKind       string
		wantRecipients []string
	}{
		{
			name:           "creating a task assigned to someone else notifies them",
			event:          models.TaskEvent{Action: "task.created", ActorID: actor, AssigneeID: ptr(asha)},
			wantKind:       "task.assigned",
			wantRecipients: []string{asha},
		},
		{
			name:  "creating a task assigned to yourself notifies nobody",
			event: models.TaskEvent{Action: "task.created", ActorID: actor, AssigneeID: ptr(actor)},
		},
		{
			name:  "creating an unassigned task notifies nobody",
			event: models.TaskEvent{Action: "task.created", ActorID: actor},
		},
		{
			name:           "setting the assignee in an edit notifies the new assignee",
			event:          models.TaskEvent{Action: "task.updated", ActorID: actor, AssigneeSent: true, AssigneeID: ptr(asha)},
			wantKind:       "task.assigned",
			wantRecipients: []string{asha},
		},
		{
			name:  "an edit that is about something else does NOT re-notify the existing assignee",
			event: models.TaskEvent{Action: "task.updated", ActorID: actor, AssigneeSent: false, AssigneeID: ptr(asha)},
		},
		{
			name:  "clearing the assignee notifies nobody",
			event: models.TaskEvent{Action: "task.updated", ActorID: actor, AssigneeSent: true, AssigneeID: nil},
		},
		{
			name:           "a comment notifies the assignee and the reporter",
			event:          models.TaskEvent{Action: "task.commented", ActorID: actor, Extra: map[string]any{"excerpt": "hi"}},
			assignee:       ptr(asha),
			reporter:       ptr(mihir),
			wantKind:       "task.commented",
			wantRecipients: []string{asha, mihir},
		},
		{
			name:           "a comment by the reporter notifies only the assignee",
			event:          models.TaskEvent{Action: "task.commented", ActorID: mihir},
			assignee:       ptr(asha),
			reporter:       ptr(mihir),
			wantKind:       "task.commented",
			wantRecipients: []string{asha},
		},
		{
			name:           "someone who is both assignee and reporter is told once",
			event:          models.TaskEvent{Action: "task.commented", ActorID: actor},
			assignee:       ptr(asha),
			reporter:       ptr(asha),
			wantKind:       "task.commented",
			wantRecipients: []string{asha},
		},
		{
			name:     "commenting on your own task notifies nobody",
			event:    models.TaskEvent{Action: "task.commented", ActorID: actor},
			assignee: ptr(actor),
			reporter: ptr(actor),
		},
		{
			name:  "a comment on a task nobody is on notifies nobody",
			event: models.TaskEvent{Action: "task.commented", ActorID: actor},
		},
		{
			name:  "moves and deletes notify nobody",
			event: models.TaskEvent{Action: "task.moved", ActorID: actor, AssigneeID: ptr(asha)},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &fakeNotificationStore{assignee: c.assignee, reporter: c.reporter}
			svc := NewNotificationService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

			svc.RecordTask(context.Background(), c.event)

			if c.wantKind == "" {
				if store.created != 0 {
					t.Fatalf("expected no notification, got %q for %v", store.kind, store.recipients)
				}
				return
			}
			if store.created != 1 || store.kind != c.wantKind {
				t.Fatalf("created %d notifications of kind %q, want 1 of %q", store.created, store.kind, c.wantKind)
			}
			if !reflect.DeepEqual(store.recipients, c.wantRecipients) {
				t.Fatalf("recipients = %v, want %v", store.recipients, c.wantRecipients)
			}
		})
	}
}

func TestTaskEventsReachesEverySink(t *testing.T) {
	a, b := &countingSink{}, &countingSink{}
	NewTaskEvents(a, b).RecordTask(context.Background(), models.TaskEvent{Action: "task.created"})
	if a.n != 1 || b.n != 1 {
		t.Fatalf("sinks got %d and %d events, want 1 each", a.n, b.n)
	}
}

type countingSink struct{ n int }

func (c *countingSink) RecordTask(context.Context, models.TaskEvent) { c.n++ }
