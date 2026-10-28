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

## Activity logs reference entities without foreign keys

activity_logs.entity_id is intentionally not a foreign key, so history survives deletion of the
entity it describes. The metadata JSONB column stores a snapshot (e.g. old and new status names)
because names can change later. Trade-off: the database cannot verify entity_id, so only the
service layer writes logs. Logs are append-only and cascade-delete only with their workspace.

## Invitation tokens are stored hashed

Same reasoning as refresh tokens: a database leak should not let anyone accept an invitation.
The raw token only appears in the emailed link.

## Invitation state is derived, not stored

Pending = not accepted, not revoked, not expired. A partial unique index allows only one pending
invitation per email per workspace. Because now() cannot appear in an index, the service must
revoke an expired pending invitation before re-inviting the same email.

## Invitations cannot grant the owner role

Ownership changes only through an existing owner promoting a member, never through a link.

## Backend layout

cmd/api/main.go only wires things together. Everything else lives under internal/ (config, database,
handlers, router), which Go prevents other modules from importing. Services and repositories will be
added when there is code for them.

## Graceful shutdown and server timeouts

The server listens for Ctrl+C / SIGTERM, stops accepting new requests, lets in-flight ones finish
(up to 10s), then closes the database pool. The http.Server sets read/write/idle timeouts so slow
clients cannot hold connections open indefinitely.

## Config from environment, .env as a dev convenience

Settings come from environment variables; backend/.env is loaded only if present and never overrides
variables that are already set. Missing or invalid config stops the app at startup with a clear error.

## Health endpoint checks the database

/health pings the database with a 2s timeout and returns 503 if it is unreachable. The handler depends
on a small Pinger interface rather than the pgx pool, so tests can inject a fake.

## Structured logging with log/slog

Text output in development, JSON in production.

## Standard response shapes

Success: {"data": ...}. Failure: {"error": {"code", "message", "request_id"}}. The frontend branches on
the stable machine-readable code, never on the human message. /health is deliberately unwrapped because
it serves infrastructure, not the frontend.

## Request IDs generated server-side

Every request gets a random ID, returned in the X-Request-ID header, included in error bodies, and
logged on every line for that request. Incoming IDs are ignored (simpler, and clients cannot inject
text into logs). Revisit if a proxy that sets its own IDs is added in front.

## Middleware order: RequestID, Logger, Recoverer

Recoverer sits inside Logger so a recovered panic becomes a normal 500 that the logger records.
Reversing them would lose the request log line for panics.

## Own Recoverer, chi's response wrapper

The recoverer is custom so panics produce our JSON error format and structured logs. The logger uses
chi's response-writer wrapper instead of a hand-written one because it preserves Flusher/Hijacker
support needed for WebSockets later.

## Log levels follow status codes; query strings are not logged

5xx logs at ERROR, 4xx at WARN, the rest at INFO. Only the URL path is logged because query strings
can carry tokens.

## CORS: explicit origin allowlist, using go-chi/cors

CORS is security-sensitive with subtle edge cases, so a maintained library is used instead of a custom
middleware. Config requires a list of exact origins: "\*" is rejected, malformed entries (trailing slash,
missing scheme) are rejected, and production refuses to start without it. This matters because the
library treats an empty list as "allow every origin". AllowCredentials is on for the refresh-token cookie,
which also requires explicit origins. CORS sits inside Logger so preflights are logged.

## Two-stage request handling: decode (400), then validate (422)

DecodeJSON handles malformed bodies; the validate package handles bad values and returns a per-field
message map. The frontend can tell "you sent something broken" from "fix this field".

## Strict JSON decoding

Bodies are capped at 1 MiB, must be application/json, reject unknown fields (guards against mass
assignment such as a client setting its own role), and reject trailing data after the JSON value.
Go's raw decoder errors are translated into safe messages so internals are not exposed.

## Hand-written validation helpers

Small, dependency-free functions. Lengths count characters, not bytes, so non-ASCII names are not
penalised. Email check is format-only. The first failure per field wins. Client-side validation (Zod) is
for user experience only; the server always validates.

## Layered architecture: handler -> service -> repository

Handlers own HTTP concerns (decoding, validation, status codes). Services own business rules and know
nothing about HTTP or SQL. Repositories own SQL and know nothing about HTTP. The service defines the
UserStore interface it needs, so tests can inject a fake. main.go is the composition root: it builds
the layers bottom-up and passes handlers to the router through a Deps struct.

## Hand-written pgx queries instead of sqlc (for now)

Writing SQL by hand makes the row-to-struct mapping visible while learning. sqlc can replace it later
without changing the service or handler layers.

## Passwords: bcrypt cost 12, 8 characters minimum, 72 bytes maximum

bcrypt is deliberately slow, and its salt is built into the stored hash. bcrypt ignores or rejects input
beyond 72 bytes, so longer passwords are rejected up front rather than silently truncated. Length is
enforced instead of composition rules. PasswordHash is tagged json:"-" so it can never be serialized.

## Email uniqueness is enforced by the database

Emails are trimmed and lowercased in the handler; the unique index on lower(email) is the real guard,
which is race-free (6 simultaneous registrations produced exactly one account). Hashing happens before
the insert, so duplicate and successful registrations take similar time. Trade-off accepted: registration
returns 409, which reveals that an email is registered.

## Never log request bodies

Registration and login bodies contain passwords; only errors and request IDs are logged.

## Two-token sessions: short-lived JWT access token + revocable refresh token

Access token: HS256 JWT, 15 minutes, carrying only the user id. It is verified by signature alone (no DB
lookup) and cannot be revoked early, which is why it is short-lived. The algorithm is pinned to HS256 so
"alg: none" and algorithm-switching attacks are refused. Refresh token: 32 random bytes, 30 days, stored
only as a SHA-256 hash in refresh_tokens so it can be revoked. Each login creates a new row, so devices
are independent sessions.

## SHA-256 for refresh tokens, bcrypt for passwords

Passwords are human-chosen and guessable, so they need a deliberately slow hash. Refresh tokens carry 256
bits of randomness and cannot be guessed, and they must be looked up by hash, which a per-hash salt (bcrypt)
would make impossible.

## Where tokens live in the browser

Refresh token: HttpOnly, SameSite=Strict, Path=/auth cookie (Secure in production), invisible to JavaScript,
never in the response body. Access token: in the response body, kept in memory by the frontend, never in
localStorage, so an XSS bug cannot read long-lived credentials. Revisit cookie SameSite/domain settings in
the deployment phase if the frontend and API end up on different sites.

## Login leaks nothing

Wrong email and wrong password return the same 401 message and take the same time (a dummy bcrypt
comparison runs for unknown emails). No password rules are enforced at login. Over-long passwords are
rejected without calling bcrypt. Responses carrying tokens are Cache-Control: no-store.

## JWT_SECRET is validated at startup

At least 32 characters, and the placeholder value is rejected, so the app cannot run with a weak or
default signing key. The secret is never committed; .env.example holds a deliberately invalid placeholder.

## Refresh token rotation with reuse detection

Every refresh revokes the old token and issues a new pair. Presenting a token that was revoked more than
10 seconds ago is treated as theft: all of that user's sessions are revoked and a warning is logged.
Within 10 seconds it is treated as a harmless race (two tabs, a double click) and just rejected.

## Atomic revoke decides races

Revoke is UPDATE ... WHERE revoked_at IS NULL and reports whether it changed a row, so of several
simultaneous refreshes with one token exactly one wins (tested: 1 success, 4 rejections).

## Logout revokes the refresh token only

Access tokens are stateless and not stored, so they cannot be cancelled; they expire within 15 minutes.
Logout is idempotent (always 204) and clears the cookie. Failed refreshes also clear the cookie.
TODO later: periodically delete old expired/revoked refresh_tokens rows.

## Protected routes via Authenticate middleware and request context

The middleware verifies the Bearer access token and stores only the user id in the request context.
Handlers get the user from the token, never from the body or URL, so one user cannot act as another.
Valid-signature tokens for deleted accounts are rejected at the point of use (user lookup returns 401).
401 means "not authenticated"; 403 (Phase 4) will mean "authenticated but not allowed".

## Rate limiting: per client address, strict on expensive endpoints

register 5/min, login 10/min, refresh 30/min, each with its own counter, applied before the handler so
blocked requests skip password hashing. Clients are identified by connection address only; forwarded
headers are ignored because they can be forged (tested). Limits are in memory and per-IP.
TODO at deployment: trusted-proxy handling, since behind a proxy all clients share one address.

## PATCH /me changes the name only

Email and password changes need confirmation / current-password checks and are deferred.
Unknown fields are rejected so a client cannot silently attempt to change other attributes.

## Frontend session model: access token in memory, refresh cookie restores it

The access token lives in a module variable, never in localStorage, so a script injected into the page
cannot read long-lived credentials. On every page load the app asks /auth/refresh to turn the HttpOnly
cookie into a session. Auth state has three values (loading, anonymous, authenticated) so a logged-in
user never sees a flash of the login page.

## One API client; silent refresh on 401, single-flight

All calls go through api(). A 401 triggers one refresh and one retry. Concurrent 401s share a single
refresh, and a request whose 401 arrives after another request already refreshed just retries with the new
token instead of refreshing again. 409 from /auth/refresh means "another tab is refreshing": wait and retry.

## Backend fix: refresh race losers must not clear the cookie

Found while designing the client. A concurrent refresh that lost the race used to receive a 401 that also
deleted the cookie, which could delete the winner's new cookie. Losers now get 409 with no cookie change.

## Hiding a page in React is not security

ProtectedRoute is for user experience. Data protection is the backend's job (Authenticate, and role checks in next phase).

## Forms validate in the browser and the server; server messages are shown per field

Zod rules mirror the backend for instant feedback (including the 72-byte password limit, counted with
TextEncoder so non-English characters are measured correctly). The server remains authoritative: its
409 and 422 responses are mapped onto the right input. Register creates the account, then logs in; if the
automatic login fails, the user is sent to /login with a notice instead of being left on a dead page.

## Reusable TextField

Label, input and error message live in one component so every form looks and behaves the same.

## Profile edits update the shared auth state

After PATCH /me succeeds, updateUser replaces the user in the auth context so every part of the UI
(greeting, future headers) updates immediately without refetching.

## Roles are per workspace; permissions are checked, not role names

A role (viewer < member < admin < owner) belongs to a membership, so one person can hold different roles in
different workspaces. The code checks permissions via permissions.Can(role, permission), and the table is one
fact per permission: the lowest role allowed. Unknown permissions and unknown roles are denied by default.
A hand-written copy of the table in the tests fails if the code's table is changed by mistake.

## Rules a permission table cannot express are enforced in services (Part 4C)

Only manage people ranked below you; only grant roles below your own (owners excepted); a workspace always
keeps at least one owner; anyone may leave unless they are the last owner.

## Workspace creation is one transaction

The workspace row and the creator's owner membership are inserted together, so a workspace can never exist
without an owner. Slugs come from the name; the unique index is the referee, and the service retries with a
random suffix on collision (tested with 8 simultaneous creations).

## API uses ids; slugs are for pretty URLs

Endpoints are keyed by workspace id (stable if a name ever changes). The frontend will use the slug in its
URLs and resolve it from the list of the user's workspaces.
TODO: cap the number of workspaces one user can own.

## Roles are looked up per request, never stored in the access token

A token cannot be changed or revoked during its 15 minutes, so a role inside it would keep working after a
demotion or removal. Roles also belong to workspaces (one person, many memberships). The token proves who
you are; the database says what you may do right now. Cost: one primary-key lookup per workspace request.
Tested: demoting, promoting and removing a user takes effect on the very next request with the same token.

## Non-members get 404, members without permission get 403

A 403 would confirm that a workspace exists. Non-members receive the exact same 404 as for a workspace that
does not exist (tested: byte-identical bodies). Malformed ids are rejected before reaching the database.

## Two small middlewares: WorkspaceAccess then RequirePermission

WorkspaceAccess loads the membership once and stores it in the request context; RequirePermission asks the
permission table. Handlers act on the authorized workspace (membership.ID), not on the raw URL parameter.
Both fail closed: a database error is a 500, and a missing membership is a refusal.

## Slugs do not change on rename; deleting a workspace cascades

Renaming keeps links working. Deleting is owner-only and removes everything inside via foreign-key cascades
(the UI must ask for confirmation).

## Rules about people live in two pure functions

CanChangeRole and CanRemove: owners manage anyone; admins only people ranked below them and only grant roles
below their own; viewers and members manage nobody. Exhaustively unit-tested (all actor x target x role
combinations). Admins therefore cannot demote themselves (not below their own rank); they can leave.

## At least one owner, enforced under a lock

Every membership change runs in a transaction that first locks the workspace row (FOR NO KEY UPDATE), so
changes in one workspace happen one at a time, then counts owners. Tested with 25 simultaneous
owner-vs-owner demotions and 25 simultaneous double-leaves: never zero owners, and exactly one request
wins each time. The lock does not block reads or other workspaces.

## Compare-and-swap on the target's role

The service reads the target's role and decides; the repository re-checks it inside the lock and answers
ErrMembershipChanged (409) if it moved in between.
Known, accepted limit: the ACTOR's role is read once at the start of the request, so someone demoted a few
milliseconds earlier can still finish one action.

## Leaving is its own action

POST /workspaces/{id}/leave with its own permission (workspace:leave, all roles). Removing yourself through
DELETE /members/{id} returns 400 pointing to it.

## Invitations: one-time link tokens, hashed at rest

A 256-bit random token is shown once, in the response that creates the invitation; only its SHA-256 hash is
stored (checked: the raw token appears nowhere in the table). No email service yet, so the inviter copies the
link. Invitations expire after 7 days and work once.

## A link alone is not enough: the accepting account's email must match

A leaked or forwarded link cannot let a stranger in. A mismatch gives 403 and leaves the link open.

## Roles in invitations follow the same "only below your own" rule

Admins invite members and viewers; owners invite admins, members and viewers; nobody can be invited as an
owner, so ownership only changes through an existing owner (see 4C).

## Re-inviting replaces the old link

Creating an invitation revokes any open one for the same email in a transaction that locks the workspace, so
at most one live link exists per email per workspace.

## Accept is one transaction on a locked invitation row

Verify open + unexpired, verify the email, verify not already a member, insert the membership, mark accepted.
Five simultaneous accepts of one link gave exactly one membership (tested).

## Public preview, uniform failures

GET /invitations/{token} needs no login (the invitee may have no account yet), is rate-limited, shows only
what the link holder already knows, and sends no-store. Unknown, expired, revoked and used links are all the
same 404. Wrong-length tokens never reach the database.
TODO (later): delete old expired invitations periodically; optional email delivery.

## Workspace data lives in TanStack Query's cache

useWorkspaces is the single source for the list; create, rename, delete and leave update the cache by hand
on success so the header dropdown, page and settings change together. The cache is cleared on login and
logout (and on session expiry) so one user never sees another's data, and logout cancels in-flight queries
first so nothing refetches while logged out.

## Session end is handled in one place

api() reports a 401 that survives a refresh attempt through a handler; the auth provider answers by clearing
the session and cache and showing the login page.

## "Where was I going" is remembered after expiry, not after logout

The login page returns you to the protected page you were heading to. After an explicit logout nothing is
remembered, so the next person to log in is not sent to the previous user's workspace URL.

## The UI hides what a role cannot do; the backend decides

permissions.ts duplicates the backend's table to decide which buttons to show. If the server still refuses
(role changed behind the page's back), its message is shown. The two tables must be kept in step; a test
comparing them is planned for the testing phase.

## Members UI offers only what the backend would accept

permissions.ts mirrors the member rules (rolesYouCanGrant, canRemoveMember, rolesYouCanInvite), so dropdowns
and buttons list only permitted choices. When the server still refuses (stale page, last owner), its message
is shown and the list is refetched.

## The invitation link is shown once, with a copy button

The token exists only in the creation response, so the UI shows the link right then. To "resend", invite the
same email again, which replaces the old link.

## The accept page works logged out, and brings you back after login or registration

/invite/:token is outside the login wall and shows a public preview. Log in and Create account pass the
current location along, so the flow ends on the invitation. Logged in with a different email shows a warning
and no accept button (the server would refuse anyway).

## Projects: permanent key, default columns created in the same transaction

The key (2-10 capital letters/digits, unique per workspace) prefixes every task id (WEB-12), so it is chosen at
creation and never changes (the update endpoint rejects it). Creating a project inserts its four default
columns (To Do, In Progress, Review, Done) in one transaction, so a project never exists without columns.

## Every project query is scoped by workspace

Get, update and delete match on project id AND workspace id, so a project can only be reached through its own
workspace. Prevents insecure direct object references (tested: another workspace's project id through this
workspace's URL gives 404).

## Reuse the permission table

project:view for listing and reading, project:manage (admin and up) for create, edit and delete. Deleting a
project cascades to its columns and tasks in the database; the UI must ask for confirmation.

## Task numbers come from a locked counter, not MAX()+1

Creating a task locks the project row and takes projects.next_task_number in the same transaction, so
simultaneous creations get distinct numbers (tested with 12 at once). Numbers are never reused, even after a
soft delete. Tasks are addressed by number in URLs (WEB-12 -> /tasks/12).

## Board order uses fractional positions, with automatic renumbering

A move gives the task the position halfway between its new neighbours, so one row changes. If two neighbours
get closer than 1e-6 the column is renumbered to 1000, 2000, ... and the move retried. The API takes
"put it directly above task N" (null = bottom) instead of raw numbers, so the server decides positions.

## PATCH tells "missing" from "null" with request.Optional

Missing leaves a field alone; null clears nullable fields (assignee, due date). Changing a task's column goes
through the move endpoint, not PATCH.

## Lists are filtered, paged and sorted through fixed choices

Sort names select fixed ORDER BY clauses (no user input in SQL text); search escapes LIKE wildcards; limit is
capped at 200; responses carry meta.total. Deleting is soft (deleted_at); everything is scoped by workspace.

## The board is built from three cached queries

A project's columns, its tasks (limit 200), and the workspace members (to show assignee initials) are fetched
with TanStack Query and combined in the page: for each column, filter tasks by status and sort by position.
Creating a task appends it to the cache, because the server always puts new tasks at the bottom of a column.

## Reusable gates and headers

WorkspaceGate handles loading, errors and "workspace not found" for any page that needs the workspace in the
URL; WorkspaceHeader shows its name, role and tabs. Pages describe only what to show once they have data.

## Keys are suggested, then yours

The new-project form suggests a key from the name until the user edits the key. Keys are permanent, so they
are shown with that warning. Boards use the key in the URL (/w/team/projects/WEB), which reads like the task
ids (WEB-12).

## Permissions again hide, never protect

"New project" needs project:manage and "+ Add task" needs task:write in the UI; the server checks both anyway.

## Moves are optimistic, with rollback and a final reload

Dropping a card updates the cached task list at once using the same "halfway between neighbours" arithmetic as
the server (board.ts: applyMove), then sends the move. On failure the previous list is restored; on completion
the tasks are reloaded so the board always converges to the server (which may have renumbered a column).

## The drop is reported as "directly above task N"

After a drag the board computes the column's final order and sends the number of the card that now follows
the moved one (null = bottom). The server decides positions, so concurrent edits cannot corrupt ordering.

## Dragging works on a working copy

While dragging, a copy of the columns (not the real data) is rearranged, so cards can preview in another column.
Dropping back where it started sends nothing.

## Pointer and keyboard are both first-class

5 px of pointer movement starts a drag (clicks still open the task); Space lifts/drops with the keyboard,
arrows move, Escape cancels, Enter opens. The open task is in the URL (?task=12), so it is linkable and
Back closes it.

## Known gaps, to do later

No UI yet for renaming/deleting projects; custom columns cannot be added or renamed (the backend only has the
four defaults); deleted tasks cannot be restored (the data is kept); the board loads up to 200 tasks.

## Comments: author edits, moderators delete, nobody rewrites someone else's words

CanEditComment (author AND still allowed to comment) and CanDeleteComment (author, or comment:moderate for
admins and owners) are small pure functions, unit-tested for every role. The routes for editing and deleting
only require task:view; the service loads the comment and decides by authorship.

## Comments survive their author

comments.author_id is emptied when an account is deleted and the API shows "Deleted user", so discussions
are not lost. Comments are hard-deleted. Everything is reached through project and workspace (tasks are
located first, then their comments), so another workspace's comment ids are unreachable.

## Comment threads cache separately from the task list

Their query key is its own branch, so reloading tasks after a drag does not reload comments.
Limit: a thread loads in full (no paging yet); a comment can be at most 5000 characters.
