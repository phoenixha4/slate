# syntax=docker/dockerfile:1.7

# The dev stage is intentionally tool-friendly. Compose Watch syncs source
# into this image and restarts `go run` on changes for a Docker-native loop.
FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
CMD ["go", "run", "./cmd/server"]

# The build stage compiles a static Linux binary. BuildKit caches Go modules
# and compile artifacts so repeat builds stay fast in local Docker and CI.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/server \
      ./cmd/server

# Distroless keeps the runtime image small and shell-free. The Go binary is
# PID 1, runs as non-root, and exposes its own healthcheck command.
FROM gcr.io/distroless/static-debian12:nonroot AS prod
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/server", "healthcheck"]
ENTRYPOINT ["/app/server"]
