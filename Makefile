DB_URL ?= postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable

.PHONY: test
test:
	go test ./...

.PHONY: integration-test
integration-test:
	SIMPLYKB_DATABASE_URL=$(DB_URL) go test ./... -run Integration

.PHONY: db-up
db-up:
	docker compose up -d

.PHONY: db-down
db-down:
	docker compose down

.PHONY: smoke
smoke:
	SIMPLYKB_DATABASE_URL=$(DB_URL) go run ./examples/quickstart
