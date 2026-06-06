# Docker Guide

Docker assets live under `docker/`. The setup is meant to make local learning easier while still showing a production-style image.

## Files

| File | Purpose |
|------|---------|
| `docker/Dockerfile` | Multi-stage dev/build/prod image |
| `docker/compose.dev.yml` | App + Postgres + Compose Watch |
| `docker/compose.prod.yml` | Production-style app container |
| `.dockerignore` | Keeps build context small and secret-free |

## Image Stages

- `dev`: Go toolchain image for `go run` and Compose Watch.
- `build`: Static Linux binary build with BuildKit cache mounts.
- `prod`: Distroless static runtime, non-root, shell-free.

## Development

```bash
just watch
```

Compose Watch:

- Syncs `cmd/`, `internal/`, and `assets/`.
- Restarts the app after synced changes.
- Rebuilds when `go.mod`, `go.sum`, or `docker/Dockerfile` changes.

## Production-Style Build

```bash
docker compose --env-file .env.example -f docker/compose.prod.yml build app
```

For a real production run, pass a real `DATABASE_URL`:

```bash
DATABASE_URL="postgres://user:pass@host:5432/todo?sslmode=require" just prod
```

## Healthcheck

The production image has no shell or curl. Docker calls:

```bash
/app/server healthcheck
```

That command calls `GET /healthz` inside the container.
