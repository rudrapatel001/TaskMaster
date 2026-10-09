-- Lets task_labels reference (task id, project id) together, so the database
-- can guarantee a label is only used on tasks in its own project.
ALTER TABLE tasks ADD CONSTRAINT tasks_id_project_unique UNIQUE (id, project_id);

CREATE TABLE comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT comments_body_check
        CHECK (length(btrim(body)) > 0 AND length(body) <= 10000)
);

-- Powers "show this task's comments, oldest first"
CREATE INDEX comments_task_idx ON comments (task_id, created_at);

CREATE TABLE labels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    color      TEXT NOT NULL DEFAULT '#6b7280',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT labels_name_check CHECK (length(btrim(name)) > 0 AND length(name) <= 50),
    CONSTRAINT labels_color_check CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
    CONSTRAINT labels_id_project_unique UNIQUE (id, project_id)
);

-- No two labels with the same name in one project, ignoring capitalization
CREATE UNIQUE INDEX labels_name_idx ON labels (project_id, lower(name));

CREATE TABLE task_labels (
    task_id    UUID NOT NULL,
    label_id   UUID NOT NULL,
    project_id UUID NOT NULL,

    PRIMARY KEY (task_id, label_id),
    CONSTRAINT task_labels_task_fk FOREIGN KEY (task_id, project_id)
        REFERENCES tasks (id, project_id) ON DELETE CASCADE,
    CONSTRAINT task_labels_label_fk FOREIGN KEY (label_id, project_id)
        REFERENCES labels (id, project_id) ON DELETE CASCADE
);

-- Powers "all tasks with this label" (the primary key already covers lookups by task)
CREATE INDEX task_labels_label_idx ON task_labels (label_id);