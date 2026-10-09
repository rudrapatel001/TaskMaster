# Technical Decisions

## UUIDs for primary keys

Chose UUIDs over auto-increment integers so IDs can't be guessed or enumerated in URLs.
Trade-off: slightly more verbose in Go and SQL.

## Storing hashes of refresh tokens

Only a hash of each refresh token is stored, so a database leak doesn't expose usable tokens.

## Roles live only in workspace_members

Ownership is not stored as an owner_id column on workspaces. The membership role is the
single source of truth, so the two can never disagree. workspaces.created_by is informational only.
A workspace can have multiple owners; "cannot remove the last owner" is enforced in application code.

## TEXT + CHECK instead of Postgres ENUM for roles

CHECK constraints are easier to change in a later migration than ENUM types.

## Different ON DELETE behavior per relationship

Deleting a workspace cascades to its memberships. Deleting a user cascades to their memberships
but keeps the workspace (created_by is set to NULL), so a team's data survives one person leaving.

## Per-project task numbers via a counter row

Each project has next_task_number. Creating a task runs UPDATE ... RETURNING inside a transaction,
which locks the project row so concurrent creates cannot get the same number. Chosen over
MAX(number)+1, which has a race condition. Numbers are never reused after soft delete.

## Fractional positions for kanban ordering

tasks.position is a DOUBLE PRECISION. Dropping a card between two neighbors takes the midpoint,
so only one row is updated. Trade-off: gaps shrink after many reorders in one spot, so a column
may occasionally need rebalancing.

## Partial indexes on live tasks

Board and "My tasks" indexes include WHERE deleted_at IS NULL, so they stay small and match
the queries the app actually runs.

## Custom statuses per project, fixed priorities

Statuses live in project_statuses (name + category of todo/in_progress/done), so teams can define
their own columns. The category lets reports know what "done" means regardless of the name.
tasks reference (status_id, project_id) through a composite foreign key, so the database guarantees
a task's status belongs to its own project. Priorities stay a fixed CHECK list because they have an
inherent order that sorting and filtering depend on; display names can be customized in the UI later.

## Assignee must be a workspace member (enforced in code)

A foreign key cannot express "assignee belongs to this workspace", so the Go service validates it.

## Labels are per project, enforced with composite foreign keys

task_labels stores project_id and references both (task_id, project_id) and (label_id, project_id),
so the database guarantees a label can only be attached to a task in the same project, same pattern
as task statuses. Trade-off: each project manages its own labels instead of sharing workspace-wide ones.
Workspace-wide labels would have required a check in application code.

## Schema changes go in new migrations

tasks needed a new UNIQUE (id, project_id) constraint for the label foreign key. Because migration 3
was already pushed and built on, the change went into migration 4 via ALTER TABLE instead of
editing migration 3.

## Comments survive their author leaving

comments.author_id is SET NULL on user deletion so discussions are preserved; the UI shows
"Deleted user". Comments are hard-deleted, and updated_at is set by Go on edit so the UI can show "(edited)".
