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
