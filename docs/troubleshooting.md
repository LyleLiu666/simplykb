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

If the database is not running, the SDK cannot create the pool or run migrations.

If you see an "address already in use" error on port `25432`, restart with a different port:

```bash
PARADEDB_PORT=35432 make smoke
```

or:

```bash
PARADEDB_PORT=35432 make db-up
PARADEDB_PORT=35432 go run ./examples/quickstart
```

## Using Plain Postgres Instead Of ParadeDB

Symptoms:

- `type "vector" does not exist`
- extension creation failures
- `pg_search` related errors

Cause:

`simplykb` expects ParadeDB because migrations require both BM25 search support and `pgvector`.

Fix:

- use the bundled [docker-compose.yml](../docker-compose.yml)
- run migrations against that database
- do not point the quickstart at a plain Postgres instance first

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
PARADEDB_PORT=35432 make integration-test
```

## Need More Context

If the issue is not covered here:

- read [README.md](../README.md) first
- confirm local setup using the bundled Compose file
- include exact error text, Go version, and database setup details when filing an issue
