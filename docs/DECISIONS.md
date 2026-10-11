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
