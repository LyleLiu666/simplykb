# Changelog

All notable changes to `simplykb` should be recorded in this file.

The format is simple on purpose:

- Added
- Changed
- Fixed
- Docs

## Unreleased

### Added

- SDK consumer integration coverage that simulates a fresh external Go module.
- Quickstart tests for local connection-string defaults.
- A production-shaped OpenAI-compatible embedder example under `examples/openai_compatible`.
- A `make verify` target for pre-release validation.
- A release guide in `RELEASING.md`.

### Changed

- Local database defaults now follow `PARADEDB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.
- CI now runs integration coverage against ParadeDB.
- ParadeDB images are pinned by digest for more repeatable local and CI behavior.

### Fixed

- `make smoke` and `make integration-test` now align with custom local ParadeDB ports.

### Docs

- README now leads with the SDK-first story and links to a production-shaped embedder example.
- Troubleshooting and contributing docs now match the current local workflow.
