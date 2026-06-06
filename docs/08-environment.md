# Environment Reference

The app reads environment variables directly. `.env` is for Docker Compose and local shell convenience only; the Go process does not load `.env` files by itself.

Start from:

```bash
cp .env.example .env
```

## App Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `APP_ENV` | No | `development` | Runtime label included in logs |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, or `error` |
| `LOG_FORMAT` | No | `json` | `json`, `text`, or `pretty`; use `pretty` for local console logs |
| `PORT` | No | `8080` | Port the Go server listens on inside the container/process |
| `HOST_PORT` | No | `8080` | Host port published by Docker Compose |
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `CORS_ALLOWED_ORIGINS` | No | unset | Comma-separated browser origins allowed by CORS |
| `READ_TIMEOUT` | No | `15s` | HTTP server read timeout |
| `WRITE_TIMEOUT` | No | `30s` | HTTP server write timeout |
| `IDLE_TIMEOUT` | No | `60s` | HTTP idle connection timeout |
| `SHUTDOWN_TIMEOUT` | No | `10s` | Graceful shutdown deadline |
| `READINESS_TIMEOUT` | No | `2s` | Database ping timeout for `/readyz` |

## Database Variables For Dev Compose

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_DB` | No | `todo` | Dev database name |
| `POSTGRES_USER` | No | `todo` | Dev database user |
| `POSTGRES_PASSWORD` | No | `todo` | Dev database password |
| `POSTGRES_HOST_PORT` | No | `5432` | Host port for the dev Postgres service |

## Production Notes

- Provide `DATABASE_URL` through your deployment environment or secret manager.
- Prefer `LOG_FORMAT=json` in production for log aggregation.
- Use `LOG_FORMAT=pretty` locally when reading logs by eye.
- Do not commit `.env`.
- Avoid `CORS_ALLOWED_ORIGINS=*` in production unless the deployment intentionally allows all browser origins.

For local learning and experiments, `.env.example` is the safest starting point.
