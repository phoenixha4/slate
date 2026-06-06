# Slate Overview

Slate is a full-stack todo application built with Go, PostgreSQL, and a vanilla-JS frontend. It is my first Go project, made for learning how the backend, database, Docker setup, and browser UI work together.

## Architecture

- One Go binary serves both the embedded frontend and same-origin REST API.
- The frontend uses ES modules and calls `/projects`, `/labels`, and `/tasks` with `fetch()`.
- PostgreSQL is required and migrations run automatically on startup.
- Docker uses a dev Compose file with Compose Watch and a production Compose file with a distroless runtime.

## Quick Start

```bash
cp .env.example .env
just watch
```

Then open <http://localhost:8080>.

## Key Files

| File | Role |
|------|------|
| `cmd/server/main.go` | Entry point, healthcheck mode, config, DB, graceful shutdown |
| `internal/config/config.go` | Environment-based app configuration |
| `internal/server/server.go` | Health routes, API routes, and static frontend |
| `internal/store/postgres.go` | PostgreSQL implementation |
| `assets/frontend/` | Embedded single-page frontend |
| `docker/Dockerfile` | Multi-stage dev/build/prod image |
| `docker/compose.dev.yml` | Local app plus PostgreSQL with Compose Watch |
| `docker/compose.prod.yml` | Production-style distroless app service |
