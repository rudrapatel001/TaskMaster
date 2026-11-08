CREATE TABLE notifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,       -- who it is for
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id     UUID REFERENCES users(id) ON DELETE SET NULL,               -- who did it
    task_id      UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}'::jsonb,                         -- snapshot for display
    read_at      TIMESTAMPTZ,                                                -- NULL while unread
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT notifications_type_check CHECK (type IN ('task.assigned', 'task.commented')),
    CONSTRAINT notifications_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object')
);

-- The list: one person's notifications, newest first.
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC, id DESC);

-- The number on the bell: only unread rows are indexed, so it stays small and fast.
CREATE INDEX notifications_user_unread_idx ON notifications (user_id) WHERE read_at IS NULL;