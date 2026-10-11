# TaskMaster

A Jira-style task and project management tool.

## Goal

Built to learn full-stack development with React, Go, and PostgreSQL,
focusing on authentication, role-based authorization, and relational data modeling.

## My angle

TaskMaster is keyboard-first and built for small teams. Most tools in this space are packed with features that students and small groups never use, so I focused on a fast, minimal workflow: create, assign, and move tasks without leaving the keyboard.

> The keyboard-first workflow is the design direction. It is not built yet; see the roadmap below.

## Status

Work in progress.

- [x] Project setup (Go backend, Vite frontend, PostgreSQL via Docker)
- [x] Database schema and migrations (users, workspaces and roles, projects, custom statuses, tasks, comments, labels, activity logs, invitations)
- [x] Backend foundation (config, DB pool, request IDs, logging, panic recovery, CORS, validation, JSON responses)
- [x] Authentication (register, login, rotating refresh tokens, logout, profile, rate limiting, React login/register pages)
- [ ] Workspaces and role-based authorization
- [ ] Projects, tasks, and kanban board
- [ ] Comments, activity feed, notifications
- [ ] Keyboard shortcuts and command palette
- [ ] Automated tests and CI
- [ ] Deployment

## What works today

- Create an account, log in, log out, and edit your display name
- Sessions survive page reloads and expired access tokens are renewed silently
- A consistent API: every response is `{"data": ...}` or `{"error": {...}}`

### Security highlights

- Passwords are hashed with **bcrypt** (cost 12); login takes the same time and gives the same error for an unknown email and a wrong password
- **Access tokens** are short-lived (15 minutes) signed JWTs, and the signing algorithm is pinned to HS256
- **Refresh tokens** are random, stored only as SHA-256 hashes, rotated on every use, and revocable. Reusing an old token ends all of that user's sessions
- The refresh token travels in an `HttpOnly`, `SameSite=Strict` cookie; the access token is kept only in memory in the browser, never in `localStorage`
- Rate limits per client address: register 5/min, login 10/min, refresh 30/min
- CORS allows only the origins you list; request bodies are size-limited and decoded strictly (unknown fields are rejected)

## Stack

- Frontend: React 19, Vite, TypeScript, Tailwind CSS, React Router, React Hook Form, Zod
- Backend: Go, chi, pgx, bcrypt, JWT
- Database: PostgreSQL

## Architecture

The backend is split into layers, each with one job:

```
request -> middleware -> router -> handler -> service -> repository -> PostgreSQL
```

| Layer      | Responsibility                                                           |
| ---------- | ------------------------------------------------------------------------ |
| middleware | request ID, logging, panic recovery, CORS, authentication, rate limiting |
| handler    | HTTP only: decode the request, validate input, choose the status code    |
| service    | business rules: hash passwords, check logins, rotate tokens              |
| repository | SQL only                                                                 |

```
taskmaster/
├── backend/
│   ├── cmd/api/          # main.go: builds and connects everything
│   ├── internal/
│   │   ├── auth/         # JWT access tokens, refresh token generation
│   │   ├── config/       # environment configuration
│   │   ├── database/     # PostgreSQL connection pool
│   │   ├── handlers/     # HTTP layer
│   │   ├── middleware/   # request ID, logging, recovery, auth, rate limit
│   │   ├── models/       # shared data types
│   │   ├── repository/   # SQL queries
│   │   ├── request/      # JSON body decoding
│   │   ├── response/     # standard JSON responses
│   │   ├── router/       # routes
│   │   ├── services/     # business logic
│   │   └── validate/     # input validation helpers
│   └── migrations/       # numbered SQL migrations
├── frontend/
│   └── src/
│       ├── auth/         # auth state, session restore, protected routes
│       ├── components/   # reusable UI pieces
│       ├── lib/          # API client
│       └── pages/        # login, register, home
├── docs/                 # schema and technical decisions
└── docker-compose.yml    # local PostgreSQL
```

## API

| Method | Path             | Auth   | Description                                                  |
| ------ | ---------------- | ------ | ------------------------------------------------------------ |
| GET    | `/health`        | no     | Server and database status                                   |
| POST   | `/auth/register` | no     | Create an account (5 requests/min)                           |
| POST   | `/auth/login`    | no     | Returns an access token and sets the refresh cookie (10/min) |
| POST   | `/auth/refresh`  | cookie | Trades the refresh cookie for new tokens (30/min)            |
| POST   | `/auth/logout`   | cookie | Revokes the refresh token and clears the cookie              |
| GET    | `/me`            | yes    | The logged-in user's profile                                 |
| PATCH  | `/me`            | yes    | Change the display name                                      |

Authenticated requests send `Authorization: Bearer <access token>`.

Responses:

```json
{ "data": { "id": "...", "email": "rudra@example.com", "name": "Rudra" } }
```

```json
{
  "error": {
    "code": "validation_failed",
    "message": "one or more fields are invalid",
    "request_id": "a1b2c3d4e5f6a7b8",
    "fields": { "email": "enter a valid email address" }
  }
}
```

## Documentation

- [Database schema](docs/SCHEMA.md)
- [Technical decisions](docs/DECISIONS.md)

## Running locally

### Prerequisites

- [Go](https://go.dev/dl/) 1.25 or newer
- [Node.js](https://nodejs.org/) 22 LTS or newer
- [Docker Desktop](https://www.docker.com/)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI:
  `go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
  (make sure your Go `bin` folder, such as `%USERPROFILE%\go\bin` on Windows, is on your `PATH`)

### 1. Start the database

```bash
docker compose up -d
```

### 2. Configure the backend

Copy `backend/.env.example` to `backend/.env` and fill it in:

| Variable               | Required      | Description                                                                             |
| ---------------------- | ------------- | --------------------------------------------------------------------------------------- |
| `DATABASE_URL`         | yes           | Must match the credentials in `docker-compose.yml`                                      |
| `JWT_SECRET`           | yes           | Random value of at least 32 characters. The server refuses to start with a placeholder  |
| `APP_ENV`              | no            | `development` (default) or `production`. Production uses JSON logs and `Secure` cookies |
| `PORT`                 | no            | Defaults to `8080`                                                                      |
| `CORS_ALLOWED_ORIGINS` | in production | Comma-separated frontend origins. Defaults to `http://localhost:5173` in development    |

Generate a random `JWT_SECRET` (PowerShell):

```powershell
$b = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
($b | ForEach-Object { $_.ToString('x2') }) -join ''
```

Never commit `backend/.env`.

### 3. Apply the migrations

From the `backend` folder:

```powershell
# PowerShell
$env:DATABASE_URL="postgres://taskmaster:taskmaster_dev_pw@localhost:5432/taskmaster?sslmode=disable"
migrate -path migrations -database $env:DATABASE_URL up
```

```bash
# macOS / Linux
export DATABASE_URL="postgres://taskmaster:taskmaster_dev_pw@localhost:5432/taskmaster?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up
```

### 4. Run the backend

From the `backend` folder:

```bash
go run ./cmd/api
```

Check it at <http://localhost:8080/health>. You should see `{"status":"ok","database":"up"}`.

### 5. Run the frontend

From the `frontend` folder:

```bash
npm install
npm run dev
```

Open <http://localhost:5173> and create an account. The API address defaults to
`http://localhost:8080`; set `VITE_API_URL` to change it. (`VITE_` variables are
public, so never put secrets in them.)

## Troubleshooting

- **`Only one usage of each socket address`**: port 8080 is already taken, usually by an older copy of the server. On Windows: `netstat -ano | findstr :8080`, then `taskkill /PID <number> /F`
- **`JWT_SECRET must be at least 32 characters`**: generate a real secret as shown above
- **`429 Too Many Requests`** while testing: the rate limiter is working. Wait a minute
- **A CORS error in the browser console**: check that `CORS_ALLOWED_ORIGINS` exactly matches the address in your browser bar (`localhost` and `127.0.0.1` are different origins), with no trailing slash

## Roadmap

1. **Workspaces and authorization**: workspaces, owner/admin/member/viewer roles, invitations
2. **Projects and tasks**: custom statuses, labels, kanban board with drag and drop, filters
3. **Collaboration**: comments, activity feed, notifications
4. **Keyboard-first workflow**: shortcuts and a command palette
5. **Quality**: backend and frontend tests, end-to-end test, CI
6. **Deployment**: Docker images, hosted demo with seed data
