# Slate Todo

Slate is my first project in Go. I built it for learning purposes, mainly to understand how a Go backend, PostgreSQL, Docker, and a small browser frontend can fit together in one practical app.

The app is a full-stack todo manager. One Go server serves both the embedded vanilla-JS frontend and the same-origin REST API from a single deployable binary.

## Quick Start

```bash
cp .env.example .env
just watch
```

Open <http://localhost:8080>. Docker Compose starts PostgreSQL, waits for it to become healthy, and runs the Go app with Compose Watch enabled for source changes.

## Docker Commands

```bash
just dev            # run the dev stack
just watch          # run the dev stack with Docker-native file watching
just prod           # run the production-style distroless app image
just docker-config  # validate dev and prod Compose files
just down           # stop Compose stacks
```

Docker files live under `docker/`:

- `docker/Dockerfile` has `dev`, `build`, and `prod` stages.
- `docker/compose.dev.yml` runs the app plus Postgres and uses Compose Watch.
- `docker/compose.prod.yml` runs the distroless production image with an external/env-provided `DATABASE_URL`.

## Environment

Use `.env.example` as the template for local settings. The app reads real environment variables directly; there is no Go `.env` loader.

Important variables:

- `DATABASE_URL` is required.
- `PORT` controls the container listen port.
- `HOST_PORT` controls the host port published by Compose.
- `LOG_LEVEL` supports `debug`, `info`, `warn`, and `error`.
- `LOG_FORMAT` supports `json`, `text`, and `pretty`; use `pretty` for local console logs.
- `CORS_ALLOWED_ORIGINS` is comma-separated; leave unset in production unless cross-origin browser calls are needed.
- Timeout values use Go duration syntax such as `15s` or `1m`.

## Local Go Run

Start PostgreSQL yourself, then run:

```bash
export DATABASE_URL="postgres://todo:todo@localhost:5432/todo?sslmode=disable"
export PORT=8080
export LOG_FORMAT=pretty
go run ./cmd/server
```

## Build And Test

```bash
just test
go build ./cmd/server
```

Health endpoints:

- `GET /healthz` checks that the process is alive.
- `GET /readyz` checks database reachability.
- `server healthcheck` is used by the distroless Docker healthcheck.

## API

API routes are served from the same origin as the frontend:

- `GET /projects`, `POST /projects`, `GET/PATCH/DELETE /projects/{id}`
- `GET /labels`, `POST /labels`, `PATCH/DELETE /labels/{id}`
- `GET /tasks`, `GET /tasks/search?q=...`, `POST /tasks`, `GET/PATCH/DELETE /tasks/{id}`

## Documentation

- [Overview](docs/00-overview.md)
- [Backend](docs/01-backend.md)
- [Database](docs/03-database.md)
- [API](docs/04-api.md)
- [Environment](docs/08-environment.md)
- [Docker](docs/09-docker.md)
- [Testing](docs/10-testing.md)
- [sqlc Notes](docs/11-sqlc.md)
- [Maintenance](docs/12-maintenance.md)
- [API Reference](api_reference.md)
- [Security](SECURITY.md)

## Note

This project is mainly for practice and learning. If you read through the code and have any tips, suggestions, or better ways to do things in Go, please feel free to share them.
