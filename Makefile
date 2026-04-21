POSTGRES_USER ?= simplykb
POSTGRES_PASSWORD ?= simplykb
POSTGRES_DB ?= simplykb
PARADEDB_PORT ?= 25432
DB_URL ?=
DB_SERVICE ?= paradedb
DB_WAIT_RETRIES ?= 30
DB_WAIT_INTERVAL ?= 2
COMPOSE_PROJECT_NAME ?= simplykb-$(shell printf '%s' "$(CURDIR)" | cksum | awk '{print $$1}')
COMPOSE = COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose
RESOLVE_DB_URL = db_url="$${DB_URL:-$$(go run ./examples/internal/exampleenv/cmd/printdburl)}"

export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB PARADEDB_PORT DB_URL

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: integration-test
integration-test:
	$(RESOLVE_DB_URL); SIMPLYKB_DATABASE_URL="$$db_url" go test ./... -run Integration -count=1 -v

.PHONY: doctor
doctor:
	$(RESOLVE_DB_URL); SIMPLYKB_DATABASE_URL="$$db_url" go run ./examples/internal/exampleenv/cmd/doctor

.PHONY: benchmark
benchmark:
	go test ./... -run '^$$' -bench '^(BenchmarkHashEmbedderMediumDocument|BenchmarkDefaultSplitterMediumDocument)$$' -benchmem -count=1

.PHONY: integration-benchmark
integration-benchmark: db-up
	$(RESOLVE_DB_URL); SIMPLYKB_DATABASE_URL="$$db_url" go test ./... -run '^$$' -bench '^BenchmarkIntegration' -benchmem -benchtime=3x -count=1

.PHONY: print-db-url
print-db-url:
	@$(RESOLVE_DB_URL); printf '%s\n' "$$db_url"

.PHONY: verify
verify: db-up
	$(MAKE) test
	$(MAKE) vet
	$(MAKE) smoke
	$(MAKE) doctor
	$(MAKE) integration-test

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
	$(RESOLVE_DB_URL); SIMPLYKB_DATABASE_URL="$$db_url" go run ./examples/quickstart

.PHONY: quickstart
quickstart: smoke
