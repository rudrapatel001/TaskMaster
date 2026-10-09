# Database schema

Current as of migration 5. Rebuild from scratch with `migrate ... up`.

```mermaid
erDiagram
    users ||--o{ refresh_tokens : "has"
    users ||--o{ workspace_members : "joins as"
    workspaces ||--o{ workspace_members : "has"
    workspaces ||--o{ projects : "contains"
    workspaces ||--o{ invitations : "sends"
    workspaces ||--o{ activity_logs : "records"
    projects ||--o{ project_statuses : "defines"
    projects ||--o{ tasks : "contains"
    projects ||--o{ labels : "defines"
    project_statuses ||--o{ tasks : "groups"
    tasks ||--o{ comments : "has"
    tasks ||--o{ task_labels : "tagged with"
    labels ||--o{ task_labels : "applied as"
    users |o--o{ tasks : "assigned to"
    users |o--o{ tasks : "reported by"
    users |o--o{ comments : "writes"
    users |o--o{ activity_logs : "performs"

    users {
        uuid id PK
        text email UK "unique ignoring case"
        text password_hash
        text name
        timestamptz created_at
        timestamptz updated_at
    }

    refresh_tokens {
        uuid id PK
        uuid user_id FK
        text token_hash UK
        timestamptz expires_at
        timestamptz revoked_at
        timestamptz created_at
    }

    workspaces {
        uuid id PK
        text name
        text slug UK
        uuid created_by FK "informational only"
        timestamptz created_at
        timestamptz updated_at
    }

    workspace_members {
        uuid workspace_id PK, FK
        uuid user_id PK, FK
        text role "owner admin member viewer"
        timestamptz joined_at
    }

    projects {
        uuid id PK
        uuid workspace_id FK
        text name
        text key "unique per workspace"
        text description
        int next_task_number
        uuid created_by FK
        timestamptz created_at
        timestamptz updated_at
    }

    project_statuses {
        uuid id PK
        uuid project_id FK
        text name "unique per project"
        text category "todo in_progress done"
        float position
        timestamptz created_at
    }

    tasks {
        uuid id PK
        uuid project_id FK
        int number "unique per project"
        text title
        text description
        uuid status_id FK "must belong to same project"
        text priority "low medium high urgent"
        uuid assignee_id FK
        uuid reporter_id FK
        date due_date
        float position
        timestamptz deleted_at "soft delete"
        timestamptz created_at
        timestamptz updated_at
    }

    comments {
        uuid id PK
        uuid task_id FK
        uuid author_id FK
        text body
        timestamptz created_at
        timestamptz updated_at
    }

    labels {
        uuid id PK
        uuid project_id FK
        text name "unique per project"
        text color
        timestamptz created_at
    }

    task_labels {
        uuid task_id PK, FK
        uuid label_id PK, FK
        uuid project_id FK "task and label must share it"
    }

    activity_logs {
        uuid id PK
        uuid workspace_id FK
        uuid actor_id FK
        text entity_type
        uuid entity_id "no FK by design"
        text action
        jsonb metadata
        timestamptz created_at
    }

    invitations {
        uuid id PK
        uuid workspace_id FK
        text email
        text role "admin member viewer"
        text token_hash UK
        uuid invited_by FK
        timestamptz expires_at
        timestamptz accepted_at
        timestamptz revoked_at
        timestamptz created_at
    }
```

The diagram leaves out the `created_by` and `invited_by` links to `users` to keep it readable. They appear as FK columns in the tables.

## Rules the database enforces

- Emails, slugs, project keys, status names, and label names can't be duplicated (case-insensitive where it matters).
- A task's status must belong to the task's own project (composite foreign key).
- A label can only be attached to a task in the same project (composite foreign keys).
- A user can be in a workspace only once, with one of four valid roles.
- Task numbers are unique per project.
- Invitations can't grant the owner role, and only one can be pending per email per workspace.
- Deleting a user keeps workspaces, tasks, comments, and logs (the user reference becomes empty).
- Deleting a workspace removes everything inside it.

## Rules the Go code must enforce

- At least one owner must always remain in a workspace, and admins can't demote owners.
- An assignee must be a member of the task's workspace.
- Creating a task uses the `next_task_number` counter inside a transaction.
- Creating a project inserts the four default statuses in the same transaction.
- A status can't be deleted while tasks (including soft-deleted ones) still use it.
- Re-inviting an email first revokes the old pending invitation.
- Accepting an invitation requires the logged-in user's email to match the invited email.
- Only the service layer writes `activity_logs`.
