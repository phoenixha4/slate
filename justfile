dev_compose := "docker/compose.dev.yml"
prod_compose := "docker/compose.prod.yml"
env_file := env_var_or_default("ENV_FILE", ".env.example")

default:
    just --list

dev:
    docker compose -f {{dev_compose}} up --build

watch:
    docker compose -f {{dev_compose}} up --watch --build

prod:
    docker compose -f {{prod_compose}} up --build

test:
    go test ./...

docker-config:
    docker compose -f {{dev_compose}} config --quiet
    docker compose --env-file {{env_file}} -f {{prod_compose}} config --quiet

down:
    docker compose -f {{dev_compose}} down
    docker compose -f {{prod_compose}} down
