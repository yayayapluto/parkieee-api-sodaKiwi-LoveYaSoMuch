.PHONY: help run run-dev seed seedfake test test-race build vet fmt tidy clean check \
	infra-up infra-down infra-logs infra-ps infra-wait infra-rebuild \
	db-up db-down db-logs

BINARY := parkieee-api
INFRA_COMPOSE_FILE ?= docker-compose.infra.yml
DOCKER_COMPOSE ?= docker compose
LEGACY_POSTGRES_CONTAINER ?= parkieee-postgres
POSTGRES_PORT ?= 5432
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= parkieee

help:
	@echo "Available targets:"
	@echo "  make run        - Run API server"
	@echo "  make run-dev    - Start full infra stack (postgres + api + cloudflared)"
	@echo "  make infra-up   - Start infra stack (docker compose)"
	@echo "  make infra-down - Stop infra stack (docker compose)"
	@echo "  make infra-rebuild - Rebuild API image + restart (after Go source change)"
	@echo "  make infra-logs - Tail infra logs"
	@echo "  make infra-ps   - Show infra services status"
	@echo "  make db-up      - Alias of infra-up"
	@echo "  make db-down    - Alias of infra-down"
	@echo "  make db-logs    - Alias of infra-logs"
	@echo "  make seed       - Seed database (idempotent)"
	@echo "  make seedfake   - Massively seed fake domain data (warning: truncates!) (override: make seedfake SCALE=2)"
	@echo "  make test       - Run all tests"
	@echo "  make test-race  - Run tests with race detector"
	@echo "  make build      - Build API binary"
	@echo "  make vet        - Run go vet"
	@echo "  make fmt        - Format Go files"
	@echo "  make tidy       - Run go mod tidy"
	@echo "  make check      - fmt + vet + test"
	@echo "  make clean      - Remove built binary"

run:
	go run ./cmd/api

run-dev: infra-up

infra-up:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "docker not found. Install/start Docker first."; \
		exit 1; \
	fi
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(LEGACY_POSTGRES_CONTAINER)$$'; then \
		echo "Removing legacy container '$(LEGACY_POSTGRES_CONTAINER)' to free port $(POSTGRES_PORT)..."; \
		docker rm -f $(LEGACY_POSTGRES_CONTAINER) >/dev/null; \
	fi
	@$(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) up -d
	@$(MAKE) infra-wait

infra-wait:
	@i=0; \
	while [ $$i -lt 40 ]; do \
		if $(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) exec -T postgres pg_isready -U $(POSTGRES_USER) -d $(POSTGRES_DB) >/dev/null 2>&1; then \
			echo "Postgres ready on localhost:$(POSTGRES_PORT)."; \
			exit 0; \
		fi; \
		i=$$((i+1)); \
		sleep 1; \
	done; \
	echo "Postgres not healthy in time. Check logs: make infra-logs"; \
	exit 1

infra-rebuild:
	@$(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) up -d --build api
	@$(MAKE) infra-wait

infra-down:
	@$(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) down

infra-logs:
	@$(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) logs -f

infra-ps:
	@$(DOCKER_COMPOSE) -f $(INFRA_COMPOSE_FILE) ps

db-up:
	@$(MAKE) infra-up

db-down:
	@$(MAKE) infra-down

db-logs:
	@$(MAKE) infra-logs

seed:
	go run ./cmd/seed

seedfake:
	go run ./cmd/seedfake $(if $(SCALE),-scale=$(SCALE))

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build -o $(BINARY) ./cmd/api

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

check: fmt vet test

clean:
	rm -f $(BINARY)
