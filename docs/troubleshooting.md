# Troubleshooting

This page covers the most common issues when trying `simplykb`.

## Database Not Reachable

Symptoms:

- connection refused
- timeout errors
- `ping database` failures

What to check:

1. Start the bundled database with `make db-up`.
2. Confirm the container is healthy with `make db-status`.
3. Confirm the local URL points to `localhost:25432` unless you changed the port.
4. Run `make doctor` after the database is up and confirm it can at least reach the database.

If the database is not running, the SDK cannot create the pool or run migrations.

If you see an "address already in use" error on port `25432`, restart with a different port:

```bash
PARADEDB_PORT=45432 make smoke
```

or:

```bash
PARADEDB_PORT=45432 make db-up
PARADEDB_PORT=45432 go run ./examples/quickstart
```

`45432` is only an example. If that port is also busy, choose any other free local port.

If you want one command that confirms the actual connection target and search path, run:

```bash
make doctor
```

`make doctor` checks the configured target database. It does not start the local ParadeDB container for you.

## Using Plain Postgres Instead Of ParadeDB

Symptoms:

- `database is missing required extension support for pg_search` style errors
- `type "vector" does not exist`
- extension creation failures
- `pg_search` related errors

Cause:

`simplykb` expects ParadeDB because migrations require both BM25 search support and `pgvector`.

Fix:

- use the bundled [docker-compose.yml](../docker-compose.yml)
- run migrations against that database
- do not point the quickstart at a plain Postgres instance first
- run `make doctor` and confirm both `pg_search` and `vector` show as `available`

## Embedding Dimension Mismatch

Symptoms:

- migration fails with an embedding dimension mismatch
- document upsert or search fails after changing embedder dimensions

Cause:

One deployment should use one embedding dimension.
If the schema was created with one dimension and the application later uses another, the contract is broken.

Fix:

- keep `Config.EmbeddingDimensions` stable for an existing deployment
- keep the embedder output dimension aligned with that config
- if you need a different dimension, use a fresh deployment or a deliberate migration strategy
- run `make doctor` and confirm the reported embedding column type still matches your configured dimension

## Changed Splitter Or Embedder But Old Chunks Still Exist

Symptoms:

- you changed the splitter or embedder implementation
- calling `UpsertDocument` with the same content did not rebuild chunks or embeddings
- search still reflects the old chunk layout

Cause:

`UpsertDocument` intentionally treats unchanged content plus unchanged retrieval-visible metadata as a cheap path.
The SDK does not try to infer splitter or embedder recipe changes from the current interfaces.

Fix:

- use `ReindexDocument` for documents that must be rebuilt under the new recipe
- do not expect a same-content `UpsertDocument` call to auto-detect recipe rollouts
- if this is a deployment-wide recipe change, plan an explicit rebuild pass in the application layer

## Search Returns No Hits

Symptoms:

- `Search` returns an empty result set

What to check:

1. Confirm `Migrate` ran successfully.
2. Confirm `UpsertDocument` completed successfully.
3. Confirm you are searching the same collection you indexed into.
4. Confirm the query is not empty.
5. Confirm the document content is meaningful for the chosen embedder.

For the quickstart, the easiest check is to rerun:

```bash
make smoke
```

If that works, the local stack is healthy.

If you still are not sure whether the problem is setup or data, run:

```bash
make doctor
```

That output tells you whether the database is reachable, whether the schema is migrated, and whether documents and chunks are present in the current schema.

## Query Cache Never Hits

Symptoms:

- `SearchDetailed(...).Diagnostics.QueryEmbeddingCacheHit` stays `false`
- `SearchDetailed(...).Diagnostics.QueryEmbeddingCacheStatus` shows `disabled`, `bypassed`, or `miss`
- you set `QueryEmbeddingCacheSize`, but repeated searches still re-embed every time

Cause:

Either the cache is cold, or the embedder is bypassing it for this request through `QueryEmbeddingCacheKeyer`.
If you set `QueryEmbeddingCacheSize` without an embedder that implements `QueryEmbeddingCacheKeyer`, `New` now fails fast instead of silently disabling the cache.

Fix:

- implement `QueryEmbeddingCacheKeyer` on your embedder if caching is safe
- include tenant, locale, model, or other request-scoped routing inputs in the returned cache key when they affect embeddings
- return `ok=false` when a request should skip the cache
- if your embedder is not safe to cache, keep `QueryEmbeddingCacheSize` at `0`
- use `QueryEmbeddingCacheStatus` when you need to tell apart `disabled`, `bypassed`, cold `miss`, and hot `hit`

## Integration Tests Are Skipped

Symptoms:

- `TestIntegrationUpsertAndSearch` shows as skipped

Cause:

The test suite only runs integration tests when `SIMPLYKB_DATABASE_URL` is set.

Fix:

```bash
make integration-test
```

If you changed the local ParadeDB port:

```bash
PARADEDB_PORT=45432 make integration-test
```

## Doctor Says The Schema Is Not Migrated Yet

Symptoms:

- `make doctor` reports `status: database reachable; schema not migrated yet`
- the migrations table is absent

Cause:

The database is reachable, but no application run has created the SDK schema yet.

Fix:

- run the quickstart with `make smoke`
- or run your own application path that calls `store.Migrate(ctx)`
- rerun `make doctor` and confirm the migrations table is now present

## Doctor Says The Schema Is Behind Current Version

Symptoms:

- `make doctor` reports `schema migration is behind current version`
- the reported `latest migration` is lower than `expected latest migration`

Cause:

The database was migrated at some earlier SDK version, but it has not yet been brought forward to the current repository schema level.

Fix:

- run an application path that calls `store.Migrate(ctx)` using the current SDK version
- rerun `make doctor`
- confirm `status: ready` and that `latest migration` matches `expected latest migration`

## Need A Reproducible Performance Baseline

Symptoms:

- you want a local performance comparison before or after a change
- the README data envelope is not enough for your rollout review

Fix:

```bash
make benchmark
make integration-benchmark
```

Use `make benchmark` for CPU-only splitter and hash-embedder baselines.

Use `make integration-benchmark` for ParadeDB-backed upsert and hybrid search baselines.

Treat those results as local comparison points, not universal production guarantees.

## Need More Context

If the issue is not covered here:

- read [README.md](../README.md) first
- confirm local setup using the bundled Compose file
- include exact error text, Go version, and database setup details when filing an issue
