# Changelog

All notable changes to `simplykb` should be recorded in this file.

The format is simple on purpose:

- Added
- Changed
- Fixed
- Docs

## Unreleased

## v0.1.0 - 2026-04-19

### Added

- SDK consumer integration coverage that simulates a fresh external Go module.
- Quickstart tests for local connection-string defaults.
- A production-shaped OpenAI-compatible embedder example under `examples/openai_compatible`.
- A `make verify` target for pre-release validation.
- A release guide in `RELEASING.md`.
- Metadata filter support in `Search`, with integration coverage and a dedicated index migration.
- Makefile-level regression tests for local database URL construction and special-character handling.

### Changed

- Local database defaults now follow `PARADEDB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.
- CI now runs integration coverage against ParadeDB.
- ParadeDB images are pinned by digest for more repeatable local and CI behavior.
- Local examples and Make targets now share the same database URL resolution path.

### Fixed

- `make smoke` and `make integration-test` now align with custom local ParadeDB ports.
- Production-shaped embedding example now streams successful embedding responses instead of truncating them at 1 MiB.
- Local default database URLs now correctly escape usernames, passwords, and database names with reserved characters.
- OpenAI-compatible example tests no longer call `t.Fatalf` from the `httptest` handler goroutine.
- Makefile database URL handling now preserves `$` and other special characters when passing URLs into smoke and integration targets.

### Docs

- README now leads with the SDK-first story and links to a production-shaped embedder example.
- Troubleshooting and contributing docs now match the current local workflow.
