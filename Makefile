POSTGRES_USER ?= simplykb
POSTGRES_PASSWORD ?= simplykb
POSTGRES_DB ?= simplykb
PARADEDB_PORT ?= 25432
DB_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(PARADEDB_PORT)/$(POSTGRES_DB)?sslmode=disable
DB_SERVICE ?= paradedb
DB_WAIT_RETRIES ?= 30
DB_WAIT_INTERVAL ?= 2
COMPOSE_PROJECT_NAME ?= simplykb-$(shell printf '%s' "$(CURDIR)" | cksum | awk '{print $$1}')
COMPOSE = COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose

export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB PARADEDB_PORT

.PHONY: test
test:
	go test ./...

.PHONY: integration-test
integration-test:
	SIMPLYKB_DATABASE_URL=$(DB_URL) go test ./... -run Integration

.PHONY: db-up
db-up:
	$(COMPOSE) up -d
	$(MAKE) db-wait

.PHONY: db-wait
db-wait:
	@container_id=$$($(COMPOSE) ps -q $(DB_SERVICE)); \
	if [ -z "$$container_id" ]; then \
		echo "paradedb container not found"; \
		exit 1; \
	fi; \
	attempt=0; \
	while [ $$attempt -lt $(DB_WAIT_RETRIES) ]; do \
		status=$$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$$container_id" 2>/dev/null || true); \
		if [ "$$status" = "healthy" ]; then \
			echo "paradedb is healthy"; \
			exit 0; \
		fi; \
		attempt=$$((attempt + 1)); \
		sleep $(DB_WAIT_INTERVAL); \
	done; \
	echo "timed out waiting for ParadeDB to become healthy"; \
	exit 1

.PHONY: db-down
db-down:
	$(COMPOSE) down

.PHONY: db-status
db-status:
	$(COMPOSE) ps

.PHONY: smoke
smoke: db-up
	SIMPLYKB_DATABASE_URL=$(DB_URL) go run ./examples/quickstart

.PHONY: quickstart
quickstart: smoke
