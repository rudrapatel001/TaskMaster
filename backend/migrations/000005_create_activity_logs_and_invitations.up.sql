CREATE TABLE activity_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type  TEXT NOT NULL,
    -- Deliberately NOT a foreign key: the log must outlive the thing it describes
    entity_id    UUID NOT NULL,
    action       TEXT NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT activity_logs_entity_type_check
        CHECK (entity_type IN ('workspace', 'member', 'invitation', 'project',
                               'status', 'label', 'task', 'comment')),
    -- Actions look like "task.created" or "task.status_changed"
    CONSTRAINT activity_logs_action_check CHECK (action ~ '^[a-z_]+\.[a-z_]+$'),
    CONSTRAINT activity_logs_metadata_check CHECK (jsonb_typeof(metadata) = 'object')
);

-- Powers the workspace feed: newest first (id breaks ties for cursor pagination)
CREATE INDEX activity_logs_workspace_idx
    ON activity_logs (workspace_id, created_at DESC, id DESC);

-- Powers the per-task (or per-anything) feed
CREATE INDEX activity_logs_entity_idx
    ON activity_logs (entity_type, entity_id, created_at DESC, id DESC);

CREATE TABLE invitations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    role         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    invited_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    accepted_at  TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT invitations_email_not_blank_check CHECK (length(btrim(email)) > 0),
    -- You cannot invite someone straight in as an owner
    CONSTRAINT invitations_role_check CHECK (role IN ('admin', 'member', 'viewer')),
    CONSTRAINT invitations_not_both_check
        CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);

-- At most one PENDING invitation per email per workspace
CREATE UNIQUE INDEX invitations_pending_idx
    ON invitations (workspace_id, lower(email))
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

-- Powers "do I have invitations waiting?"
CREATE INDEX invitations_email_idx ON invitations (lower(email));