# TaskMaster

A Jira-style task and project management tool.

## Goal

Built to learn full-stack development with React, Go, and PostgreSQL,
focusing on authentication, role-based authorization, and relational data modeling.

## My angle

TaskMaster is keyboard-first and built for small teams. Most tools in this space are packed with features that students and small groups never use, so I focused on a fast, minimal workflow: create, assign, and move tasks without leaving the keyboard.

## Status

Work in progress. Done so far:

- [x] Project setup (Go backend, Vite frontend, PostgreSQL via Docker)
- [x] Database schema and migrations (users, workspaces and roles, projects, custom statuses, tasks, comments, labels, activity logs, invitations)
- [ ] Backend foundation (config, DB pool, middleware)
- [ ] Authentication
- [ ] Workspaces and role-based authorization
- [ ] Projects, tasks, and kanban board
- [ ] Comments, activity feed, notifications

## Stack

- Frontend: React, Vite, TypeScript, Tailwind
- Backend: Go, Chi, pgx
- Database: PostgreSQL

## Documentation

- [Database schema](docs/SCHEMA.md)
- [Technical decisions](docs/DECISIONS.md)

## Running locally

### Prerequisites

- [Go](https://go.dev/dl/)
- [Node.js](https://nodejs.org/) (LTS)
- [Docker Desktop](https://www.docker.com/)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI:
  `go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

### 1. Start the database

```bash
docker compose up -d
```

### 2. Configure the backend

Copy `backend/.env.example` to `backend/.env` and set the values. The database
credentials must match those in `docker-compose.yml`.

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

Check it at <http://localhost:8080/health>.

### 5. Run the frontend

From the `frontend` folder:

```bash
npm install
npm run dev
```

Open the URL Vite prints (usually <http://localhost:5173>).
