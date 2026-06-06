# Architecture Overview

This document describes the current Slate Todo application architecture. The project is intentionally compact because it is my first Go project and was built for learning.

## 1. Shape Of The App

- The Go server is the single runtime process.
- It serves embedded frontend assets and root-level REST API routes from one binary.
- The browser frontend still calls the Go API with same-origin `fetch()` requests.
- PostgreSQL is the only external runtime dependency.
- Docker assets live in `docker/`, with separate dev and production Compose files.

For this app, serving the frontend from Go is the ideal default: the frontend has no build step, deployment stays simple, and same-origin API calls avoid CORS complexity. A future split only makes sense if the frontend needs an independent build/deploy pipeline or CDN hosting.

## 2. Repository Structure

```text
.
├── .env.example
├── justfile
├── go.mod
├── docker/
│   ├── Dockerfile
│   ├── compose.dev.yml
│   └── compose.prod.yml
├── cmd/server/main.go
├── internal/
│   ├── config/
│   ├── db/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── server/
│   └── store/
├── assets/
│   ├── embed.go
│   └── frontend/
└── docs/
```

## 3. Runtime Flow

1. `cmd/server/main.go` handles the optional `server healthcheck` mode.
2. Normal startup loads env config, including log level, log format, port, CORS origins, and timeouts.
3. `internal/db` creates a pgx pool, verifies connectivity, and runs idempotent DDL.
4. `internal/store` wraps the database pool behind the `Store` interface.
5. `internal/server` registers health, API, and embedded frontend routes.
6. Middleware adds panic recovery, structured request logging, and config-driven CORS.

## 4. API And Health

Routes are same-origin and root-relative:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Process liveness |
| GET | `/readyz` | Database readiness |
| GET | `/projects` | List projects with active task counts |
| POST | `/projects` | Create a project |
| GET | `/projects/{id}` | Get one project |
| PATCH | `/projects/{id}` | Update a project |
| DELETE | `/projects/{id}` | Delete a project |
| GET | `/labels` | List labels |
| POST | `/labels` | Create a label |
| PATCH | `/labels/{id}` | Update a label |
| DELETE | `/labels/{id}` | Delete a label |
| GET | `/tasks` | List tasks with optional filters |
| GET | `/tasks/search?q=...` | Search tasks by title and notes |
| POST | `/tasks` | Create a task |
| GET | `/tasks/{id}` | Get one task with labels and subtasks |
| PATCH | `/tasks/{id}` | Update a task |
| DELETE | `/tasks/{id}` | Delete a task and subtasks |

## 5. Docker

`docker/Dockerfile` has three stages:

- `dev`: Go toolchain image for `go run` and Compose Watch.
- `build`: static binary build with BuildKit cache mounts.
- `prod`: distroless static runtime, non-root, shell-free, app binary only.

`docker/compose.dev.yml` runs Postgres and the app, and uses Compose Watch to sync/restart on `cmd/`, `internal/`, and `assets/` changes. `docker/compose.prod.yml` runs only the production app image and expects `DATABASE_URL` from the environment.

## 6. Testing

```bash
go test ./...
docker compose -f docker/compose.dev.yml config --quiet
docker compose --env-file .env.example -f docker/compose.prod.yml config --quiet
```
