# Changelog

All notable changes to `simplykb` should be recorded in this file.

The format is simple on purpose:

- Added
- Changed
- Fixed
- Docs

## v0.2.1 - 2026-04-23

### Added

- `examples/ingest_dir` as a recursive local directory ingestion example that defaults to hash embeddings and can switch to an OpenAI-compatible embeddings API through environment variables.
- `examples/internal/exampleembed` as a shared example-only embedder helper so local and provider-backed example flows reuse the same configuration and validation path.

### Changed

- `examples/openai_compatible` now reuses the shared example embedder helper instead of carrying a separate copy of the OpenAI-compatible embedding client.
- The OpenAI-compatible example loader now rejects conflicting `SIMPLYKB_EMBEDDER_PROVIDER` overrides instead of silently drifting to another embedder.
- Directory ingestion now skips empty or non-UTF-8 files by default and supports `SIMPLYKB_INGEST_STRICT=true` for fail-fast runs.
- Completed planning material now lives under `docs/plans/archive/` so active guidance stays separate from historical design notes.

### Fixed

- The documented `examples/openai_compatible` command is now self-contained even when `SIMPLYKB_EMBEDDER_PROVIDER` is already set in the shell.
- `examples/ingest_dir` now emits valid `file://` source URIs for Windows drive paths and UNC paths without corrupting legitimate Unix file names that contain a backslash character.

### Docs

- README now states which capabilities intentionally stay outside the core SDK and points readers to the new local directory ingestion example.
- The docs index now links directly to the archived plan index instead of keeping a separate active-plan shim after the current planning work was completed.

## v0.2.0 - 2026-04-23

### Added

- `ReindexDocument` as an explicit caller-driven full rebuild path for splitter or embedder rollout scenarios where content is unchanged.
- `SearchDetailed` with opt-in retrieval diagnostics while keeping `Search` as the simple compatibility path.
- `Config.QueryEmbeddingCacheSize` for an optional per-store query embedding cache on repeated vector or hybrid searches.
- `QueryEmbeddingCacheKeyer` so embedders can opt into caching with an explicit, context-aware cache key or bypass rule.
- `SearchDiagnostics.QueryEmbeddingCacheStatus` so callers can distinguish disabled, bypassed, miss, hit, and not-applicable states.
- Write-path integration coverage for `noop`, metadata-only refresh, and forced reindex behavior.
- Search integration coverage for diagnostics, cache-enabled behavior, cache-disabled behavior, and concurrent warm-cache reads.
- A repeated-upsert integration benchmark for the unchanged-document path.
- A Makefile preflight check that fails early when the chosen local ParadeDB port is already in use.

### Changed

- `UpsertDocument` can now short-circuit unchanged documents instead of always re-splitting, re-embedding, and rewriting all chunks.
- Metadata-only document updates now refresh duplicated chunk metadata in place without rebuilding chunks or embeddings.
- `Search` now delegates to the same underlying path as `SearchDetailed`, so hit ordering stays aligned across both entrypoints.
- Query embedding cache configuration now fails fast unless the embedder explicitly implements `QueryEmbeddingCacheKeyer`.
- Local integration benchmarks now show the intended fast paths clearly: unchanged-document upserts are materially cheaper than full rewrites, and cached vector searches are materially cheaper than uncached ones.

### Fixed

- Local Docker startup now reports a clearer error before `docker compose up` when `PARADEDB_PORT` is already occupied.

### Docs

- README and troubleshooting now state that unchanged documents do not automatically refresh after splitter or embedder recipe changes.
- README and troubleshooting now document `ReindexDocument` as the required rebuild path for recipe-change rollouts.
- README and troubleshooting now document `QueryEmbeddingCacheKeyer` as the required contract for enabling query embedding cache.
- Release notes now call out that `UpsertDocument` no longer serves as an implicit rebuild path for unchanged documents after recipe changes.

## v0.1.1 - 2026-04-21

### Added

- A dedicated GitHub Actions integration workflow under `.github/workflows/integration.yml`.
- GitHub issue template configuration with support links to troubleshooting and contribution guidance.
- A clearer migration preflight error when the database does not expose required `pg_search` or `vector` extension support.
- A `make doctor` command plus database diagnostics collection for extension support, schema state, and indexed row counts.
- Integration coverage for upgrading an older schema state to the current migration set without losing data.
- Integration coverage for the OpenAI-compatible example against a real ParadeDB schema and a mock embeddings endpoint.
- Reproducible `make benchmark` and `make integration-benchmark` commands for CPU and ParadeDB-backed baseline measurements.

### Changed

- The default `CI` workflow is now a baseline unit-and-vet workflow, with integration coverage split into its own workflow.
- README now shows a sample quickstart output shape and a license badge.
- `make verify` now runs `make doctor` as part of the repo-level acceptance path.
- The integration workflow now runs the diagnostics command before integration tests.

### Docs

- README now shows a separate integration status badge and a short support expectations section.
- RELEASING now records the intended meaning of the `v0.1.0` milestone.
- RELEASING now includes explicit compatibility expectations and a `v0.1.0` milestone checklist.
- README and troubleshooting now document runtime diagnostics and reproducible benchmark entrypoints.

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
