CREATE TABLE projects (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    key              TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    next_task_number INTEGER NOT NULL DEFAULT 1,
    created_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT projects_key_format_check CHECK (key ~ '^[A-Z][A-Z0-9]{1,9}$'),
    CONSTRAINT projects_name_not_blank_check CHECK (length(btrim(name)) > 0),
    CONSTRAINT projects_next_task_number_check CHECK (next_task_number >= 1),
    CONSTRAINT projects_workspace_key_unique UNIQUE (workspace_id, key)
);

CREATE TABLE project_statuses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL,
    position   DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT project_statuses_name_not_blank_check CHECK (length(btrim(name)) > 0),
    CONSTRAINT project_statuses_category_check
        CHECK (category IN ('todo', 'in_progress', 'done')),
    -- Lets tasks reference (id, project_id) together, see tasks_status_fk below
    CONSTRAINT project_statuses_id_project_unique UNIQUE (id, project_id)
);

-- No two statuses with the same name in one project, ignoring capitalization
CREATE UNIQUE INDEX project_statuses_name_idx
    ON project_statuses (project_id, lower(name));

CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number      INTEGER NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status_id   UUID NOT NULL,
    priority    TEXT NOT NULL DEFAULT 'medium',
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date    DATE,
    position    DOUBLE PRECISION NOT NULL DEFAULT 0,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT tasks_project_number_unique UNIQUE (project_id, number),
    CONSTRAINT tasks_title_not_blank_check CHECK (length(btrim(title)) > 0),
    CONSTRAINT tasks_priority_check
        CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    -- The status must belong to the SAME project as the task
    CONSTRAINT tasks_status_fk FOREIGN KEY (status_id, project_id)
        REFERENCES project_statuses (id, project_id)
);

-- Powers the kanban board: "live tasks in this project, grouped by status, in order"
CREATE INDEX tasks_board_idx ON tasks (project_id, status_id, position)
    WHERE deleted_at IS NULL;

-- Powers the "My tasks" dashboard
CREATE INDEX tasks_assignee_idx ON tasks (assignee_id)
    WHERE deleted_at IS NULL;