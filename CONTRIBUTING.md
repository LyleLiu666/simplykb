# Contributing To simplykb

Thanks for helping improve `simplykb`.

This project is intentionally narrow.
The best contributions make it clearer, safer, and easier to use without turning it into a larger platform.

## Project Purpose

`simplykb` is a low-entropy Go SDK for text-first recall workloads on ParadeDB plus `pgvector`.

The main promise is:

- one database
- one embedded SDK
- predictable migrations
- deterministic document writes
- simple hybrid recall

Please keep that promise in mind when proposing or implementing changes.

## Local Setup

Start the local database:

```bash
docker compose up -d
```

The default local connection string is:

```bash
postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable
```

## Run Tests

Run the baseline test suite:

```bash
go test ./...
go vet ./...
```

Run integration tests:

```bash
SIMPLYKB_DATABASE_URL=postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable \
go test ./... -run Integration
```

Run the quickstart example:

```bash
SIMPLYKB_DATABASE_URL=postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable \
go run ./examples/quickstart
```

## When To Add Unit Tests

Add or update unit tests when a change affects:

- request validation
- chunk splitting behavior
- ranking logic
- vector formatting
- normalization rules
- helper behavior with clear in-memory inputs and outputs

## When To Add Integration Tests

Add or update integration tests when a change affects:

- migrations
- SQL behavior
- schema assumptions
- document upsert or delete flow
- search behavior
- contract-level normalization

If a change touches setup, migrations, or retrieval behavior, run the integration suite before considering the change done.

## Keep Scope Narrow

Changes are usually welcome when they:

- improve correctness
- improve docs and onboarding
- strengthen tests
- clarify errors and contracts
- keep the public API small
- improve realistic production guidance

Changes should be discussed before implementation when they add:

- new platform layers
- tenant auth or ACL logic
- queues or workers
- dashboards
- raw query DSLs
- broad provider-specific abstractions

## Architecture Guardrails

Please protect the low-entropy design:

- Postgres stays the only durable system.
- ParadeDB and `pgvector` stay inside the same database boundary.
- The SDK remains embedded in Go applications.
- New abstractions should be justified by repeated use, not speculation.
- If a feature would make `simplykb` look like a platform, treat that as a design discussion first.

## Pull Request Expectations

Keep pull requests focused and easy to review.

A strong pull request usually includes:

- a short explanation of the change
- tests for behavior changes
- docs updates when public behavior changes
- notes about quickstart or example impact
- explicit mention of integration verification when contracts changed

## Public-Change Checklist

Before opening or merging a PR that affects onboarding, setup, or public contracts, confirm:

- code updated
- tests updated
- docs updated
- example impact checked
- release note impact checked

## Review Notes

When touching behavior guarded by existing integration tests, keep these protections intact:

- migration must fail early on embedding dimension drift
- upsert must reject empty splitter output
- delete must normalize `documentID` consistently
- schema should avoid redundant index noise
