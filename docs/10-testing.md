# Testing Guide

The tests are intentionally focused on the parts that are easiest to validate while learning: handlers, builds, Docker config, and a small manual smoke test.

## Unit Tests

```bash
go test ./...
```

Handler tests use an in-memory implementation of `store.Store`; they do not require PostgreSQL.

## Build Checks

```bash
go build ./cmd/server
```

## Docker Checks

```bash
just docker-config
docker compose -f docker/compose.dev.yml build app
docker compose --env-file .env.example -f docker/compose.prod.yml build app
```

## Manual Smoke Test

```bash
cp .env.example .env
just watch
```

Then verify:

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
curl -fsS http://localhost:8080/projects
curl -fsSI http://localhost:8080/
```

Expected behavior:

- `/healthz` returns `{"status":"ok"}`.
- `/readyz` returns `{"status":"ready"}` when PostgreSQL is reachable.
- `/projects` includes the seeded Inbox project.
- `/` returns the embedded frontend HTML.
