include .env
-include .env.prod
export

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## start: stop local postgres, start docker db, run migrations and start api
.PHONY: start
start:
	@sudo systemctl stop postgresql
	@docker compose up -d db
	@echo "Waiting for database..."
	@until docker exec bondrota-db pg_isready -U postgres; do sleep 1; done
	@$(MAKE) migration/up
	@air

## build: build the application
.PHONY: build
build:
	@go build -o bin/bondrota ./cmd/main.go

## infra/up: start only the database container
.PHONY: infra/up
infra/up:
	@docker compose up -d db
	@echo "Waiting for database..."
	@until docker exec bondrota-db pg_isready -U postgres; do sleep 1; done
	@echo "Database ready!"

## infra/down: stop the database container
.PHONY: infra/down
infra/down:
	@docker compose down

## up: start all docker containers
.PHONY: up
up:
	@docker compose up -d

## reset: stop containers and remove volumes
.PHONY: reset
reset:
	@docker compose down -v

## logs: tail api logs
.PHONY: logs
logs:
	@docker compose logs -f app

## db: connect to the database via psql
.PHONY: db
db:
	@docker exec -it bondrota-db psql -U postgres -d bondrota_db

## migration/new name=$1: create a new goose migration
.PHONY: migration/new
migration/new:
	@echo "Creating migration $(name)..."
	@goose -dir internal/db/migrations create $(name) sql

## migration/fix: convert timestamp migrations to sequential numbering
.PHONY: migration/fix
migration/fix:
	@echo "Formatting migrations..."
	@goose -dir internal/db/migrations fix

## migration/up: apply all migrations locally
.PHONY: migration/up
migration/up:
	@goose -dir internal/db/migrations postgres "$(DATABASE_URL)" up

## migration/up/prod: apply all migrations in production
.PHONY: migration/up/prod
migration/up/prod:
	@goose -dir internal/db/migrations postgres "$(PROD_DATABASE_URL)" up

## migration/down: rollback last migration
.PHONY: migration/down
migration/down:
	@goose -dir internal/db/migrations postgres "$(DATABASE_URL)" down

## migration/status: show migration status locally
.PHONY: migration/status
migration/status:
	@goose -dir internal/db/migrations postgres "$(DATABASE_URL)" status

## migration/status/prod: show migration status in production
.PHONY: migration/status/prod
migration/status/prod:
	@goose -dir internal/db/migrations postgres "$(PROD_DATABASE_URL)" status

## test: run unit tests (no DB required)
.PHONY: test
test:
	@go test ./... -count=1

## test/integration: run integration tests (requires running DB)
.PHONY: test/integration
test/integration:
	@go test ./... -tags integration -count=1 -v