# Maintenance Guide

This checklist keeps the learning project easy to revisit and improve without losing track of the basics.

## Dependency Updates

Check outdated Go dependencies:

```bash
go list -u -m all
```

Update a dependency:

```bash
go get example.com/module@latest
go mod tidy
go test ./...
```

## Docker Base Images

Rebuild images regularly to pick up base image updates:

```bash
docker compose -f docker/compose.dev.yml build --pull app
docker compose --env-file .env.example -f docker/compose.prod.yml build --pull app
```

## Database Changes

Schema lives in `internal/db/db.go`. Changes should be idempotent because migrations run on every startup.

For schema changes:

- Update the SQL.
- Update `docs/03-database.md`.
- Add or update API/store tests where practical.
- Test against a fresh database and an existing database.

## Release Checklist

- `go test ./...`
- `go build ./cmd/server`
- `just docker-config`
- Build dev and prod Docker images.
- Confirm `.env.example` includes all required settings.
