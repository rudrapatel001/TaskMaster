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
