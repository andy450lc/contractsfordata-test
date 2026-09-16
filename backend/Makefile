SHELL := /bin/bash

MIGRATIONS_DIR := internal/migrations
DEFAULT_DB_URL := postgres://sow:sow@localhost:5432/sow?sslmode=disable

# Sources .env with shell semantics before a recipe that needs it.
# Quoted values stay intact. The app itself loads .env via godotenv, so
# run and dev need no sourcing here.
ENV_PREFIX := set -a; [ -f .env ] && . ./.env; set +a; : "$${DATABASE_URL:=$(DEFAULT_DB_URL)}";

.PHONY: run dev build test lint db-up db-down migrate-up migrate-down migrate-force sqlc-generate oapi-generate

run:
	go run ./cmd/app

# Live-reload dev mode with air, configured in .air.toml. Dev convenience only.
dev:
	go tool air

build:
	go build -o tmp/app ./cmd/app

# Testcontainers' Ryuk reaper fails to start in sandboxed local
# environments and the suite then hangs before the first test. Local
# runs disable the reaper. Container cleanup relies on suite teardown.
test: export TESTCONTAINERS_RYUK_DISABLED=true
test:
	go test -race ./...

lint:
	golangci-lint run

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose stop postgres

migrate-up:
	@$(ENV_PREFIX) go tool goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" up

# Destructive targets are guarded to STAGE=dev.
migrate-down:
	@$(ENV_PREFIX) if [ "$${STAGE:-dev}" != "dev" ]; then echo "migrate-down is destructive and allowed only with STAGE=dev"; exit 1; fi; \
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" down

migrate-force:
	@$(ENV_PREFIX) if [ "$${STAGE:-dev}" != "dev" ]; then echo "migrate-force is destructive and allowed only with STAGE=dev"; exit 1; fi; \
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" up-to $(VERSION)

sqlc-generate:
	go tool sqlc generate

oapi-generate:
	go tool oapi-codegen -config oapi-codegen.yaml ../contracts/openapi.yaml
