# simplykb

[![CI](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml/badge.svg)](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml)
[![Integration](https://github.com/LyleLiu666/simplykb/actions/workflows/integration.yml/badge.svg)](https://github.com/LyleLiu666/simplykb/actions/workflows/integration.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`simplykb` is an embedded Go SDK for adding hybrid recall to a Go service with one ParadeDB database.

It is built for teams that want BM25 plus vector recall inside Postgres, without standing up a separate search service, queue, or worker fleet.

## North Star

The north star for `simplykb` is simple:

- a Go developer adds a small library to an existing service
- the team starts one local ParadeDB with Docker for development
- the service calls `Migrate`, `UpsertDocument`, and `Search`
- retrieval works without introducing another platform boundary

If `simplykb` feels like "another product to deploy beside my app", the project has drifted.

If it feels like "a small Go SDK with one database dependency", the project is on track.

## What Docker Is For

`simplykb` is not a Docker product.
It is a Go SDK.

This repository ships Docker Compose because the SDK depends on ParadeDB, and Docker is the lightest way for most developers to get a correct local database with BM25 plus vector support.

In production, your Go application still embeds the SDK directly and connects to a ParadeDB instance you manage.

## Best Fit

`simplykb` is a good fit if you want:

- one ParadeDB-backed database as the only durable system
- one embedded Go SDK instead of another HTTP service
- synchronous and deterministic document upsert
- built-in chunking and stable chunk ids
- a small public API that is easy to reason about

## Not A Fit

`simplykb` is probably not the right tool if you need:

- built-in PDF parsing, OCR, or a document ingestion platform inside the SDK
- multi-tenant auth or ACL logic inside the SDK
- distributed writes or a large async indexing pipeline inside the SDK
- a hosted search product with dashboards and workflow engines
- a broad RAG framework with many provider-specific features

## Current Shape

This repository intentionally targets a narrow but production-usable SDK shape:

- single ParadeDB database
- single embedding model per deployment
- embedded SDK, not platform product
- predictable schema and migration path
- explicit limits instead of hidden magic

Already included:

- versioned schema migration with advisory lock
- legacy-schema upgrade regression coverage in integration tests
- document upsert and delete
- no-op upsert for unchanged content plus unchanged retrieval-visible metadata
- metadata-only refresh without re-splitting or re-embedding
- default chunk splitter
- stable chunk ids
- explicit `ReindexDocument` for splitter or embedder rollout rebuilds
- hybrid recall with reciprocal rank fusion
- opt-in `SearchDetailed` diagnostics alongside the existing `Search` API
- optional per-store query embedding cache for repeated vector or hybrid queries when the embedder opts in safely
- simple metadata filter support in `Search`
- local Docker Compose for ParadeDB
- integration tests, including a provider-shaped OpenAI-compatible path
- `make doctor` runtime diagnostics for database, extensions, schema, and counts
- reproducible `make benchmark` and `make integration-benchmark` baselines
- quickstart example

## What Stays Outside The Core SDK

Some capabilities are reasonable parts of a system built with `simplykb`, but they intentionally stay outside this core package.

Use adjacent application code or services for:

- PDF parsing or OCR before calling `UpsertDocument`
- async queue, worker, or cron flows that call `UpsertDocument` or `ReindexDocument`
- reranking after `Search` when your workload proves the extra latency and cost are worth it

Treat the following as out of scope for the core SDK unless the product definition changes:

- ACL and multi-tenant policy enforcement inside the SDK
- dashboard or workflow-product surfaces
- distributed write paths

If you need those as built-in core features, choose another system or add them as a separate layer above `simplykb`.

## What Success Looks Like

A healthy first evaluation looks like this:

1. Start ParadeDB locally.
2. Run the quickstart.
3. See three `indexed doc-...` lines.
4. See a `Top hits:` section with at least one result.
5. Copy the SDK shape into your own Go service and swap `HashEmbedder` for a real embedder.

The goal is not to make you adopt a new platform.
The goal is to help you add retrieval to a Go codebase with one SDK and one database.

## Quick Start

### 1. Fastest repo evaluation

```bash
make smoke
```

This command:

- starts the bundled ParadeDB database
- waits until the database health check passes
- runs the quickstart example against the default local URL

Example output shape:

```text
indexed doc-bm25 with 1 chunks
indexed doc-vector with 1 chunks
indexed doc-hybrid with 1 chunks

Top hits:
- doc-bm25 chunk=0 score=...
```

Treat that block as an example shape, not an exact byte-for-byte expectation.
The stable success signals are:

- three successful `indexed doc-...` lines
- a `Top hits:` section
- at least one non-empty hit result
- no requirement to match exact scores or snippet text byte-for-byte

If port `25432` is already in use on your machine, override the local ParadeDB port:

```bash
PARADEDB_PORT=45432 make smoke
```

`45432` is only an example. If that port is also busy, pick any free local port.

### 2. Manual path

If you want to see each step explicitly:

```bash
make db-up
make db-status
go run ./examples/quickstart
```

The quickstart example defaults to:

```bash
postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable
```

The quickstart also reads `PARADEDB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` when building its local default connection string.

Set `SIMPLYKB_DATABASE_URL` only if you want to bypass that local default behavior:

```bash
PARADEDB_PORT=45432 make db-up
PARADEDB_PORT=45432 go run ./examples/quickstart
```

This repository expects ParadeDB, not plain Postgres.

### 3. Embed It In Your Service

`simplykb` is meant to run inside your Go service, not behind another HTTP hop.

The basic integration shape is:

```go
databaseURL := os.Getenv("SIMPLYKB_DATABASE_URL")
if databaseURL == "" {
    databaseURL = "postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable"
}

store, err := simplykb.New(ctx, simplykb.Config{
    DatabaseURL:         databaseURL,
    DefaultCollection:   "docs",
    EmbeddingDimensions: 256,
    MaxConns:            8,
    Embedder:            simplykb.NewHashEmbedder(256), // Demo-only. Use a real provider in production.
})
if err != nil {
    panic(err)
}
defer store.Close()

if err := store.Migrate(ctx); err != nil {
    panic(err)
}

_, err = store.UpsertDocument(ctx, simplykb.UpsertDocumentRequest{
    DocumentID: "doc-1",
    Title:      "BM25 notes",
    Content:    "BM25 is still strong for exact names, short queries, and logs.",
})
if err != nil {
    panic(err)
}

hits, err := store.Search(ctx, simplykb.SearchRequest{
    Query: "exact names and logs",
    Limit: 5,
})
if err != nil {
    panic(err)
}

for _, hit := range hits {
    fmt.Println(hit.DocumentID, hit.ChunkID, hit.Score, hit.Snippet)
}
```

The caller injects the `Embedder`.
That keeps provider choice outside the SDK core.

### 4. Production Example Shape

If you want a real provider-shaped example instead of `HashEmbedder`, see:

- [examples/openai_compatible/main.go](examples/openai_compatible/main.go)

That example keeps provider-specific code outside the SDK core and shows one practical pattern for an OpenAI-compatible embeddings API using environment variables:

- `SIMPLYKB_EMBEDDING_URL`
- `SIMPLYKB_EMBEDDING_API_KEY` when your provider requires bearer auth
- `SIMPLYKB_EMBEDDING_MODEL`
- `SIMPLYKB_EMBEDDING_DIMENSIONS`

Run it like this after ParadeDB is up:

```bash
SIMPLYKB_EMBEDDER_PROVIDER=openai_compatible \
SIMPLYKB_EMBEDDING_URL=https://your-provider.example/v1/embeddings \
SIMPLYKB_EMBEDDING_MODEL=... \
SIMPLYKB_EMBEDDING_DIMENSIONS=1536 \
go run ./examples/openai_compatible
```

If your provider requires auth, also set `SIMPLYKB_EMBEDDING_API_KEY`.

That example shape is also exercised by an integration test against a real ParadeDB schema and a mock OpenAI-compatible embeddings endpoint, so it is not just documentation-only.

### 5. Local Directory Ingestion Example

If you want a small ingestion entrypoint without committing to one embedding provider up front, see:

- [examples/ingest_dir/main.go](examples/ingest_dir/main.go)

That example:

- walks a local directory recursively
- ingests common text-like files such as `.md`, `.txt`, `.json`, `.yaml`, and `.html`
- uses `hash` embeddings by default, so it can run without any external embedding API
- can switch to an OpenAI-compatible embeddings endpoint later through environment variables instead of code changes
- skips empty or non-UTF-8 files by default, and can fail fast with `SIMPLYKB_INGEST_STRICT=true`

Hash-only local run:

```bash
SIMPLYKB_SOURCE_DIR=./docs \
go run ./examples/ingest_dir
```

Switch to an OpenAI-compatible embeddings API when you have one:

```bash
SIMPLYKB_SOURCE_DIR=./docs \
SIMPLYKB_EMBEDDER_PROVIDER=openai_compatible \
SIMPLYKB_EMBEDDING_URL=https://your-provider.example/v1/embeddings \
SIMPLYKB_EMBEDDING_MODEL=... \
SIMPLYKB_EMBEDDING_DIMENSIONS=1536 \
go run ./examples/ingest_dir
```

Add `SIMPLYKB_EMBEDDING_API_KEY` too if that endpoint requires bearer auth.

## Production Notes

`HashEmbedder` exists for tests, local demos, and zero-cost smoke verification.
Do not treat it as a production embedding strategy.

Production deployments should:

- plug in a real provider through the `Embedder` interface
- keep one embedding dimension per deployment
- avoid changing embedding dimensions against an already-migrated schema
- treat `simplykb` as a narrow embedded SDK, not a platform boundary
- leave `QueryEmbeddingCacheSize` at `0` unless you want an in-process query cache
- if you set `QueryEmbeddingCacheSize`, your embedder must implement `QueryEmbeddingCacheKeyer` or `New` will fail fast
- make that cache key include tenant, locale, model, or any other request-scoped routing input your embedder actually uses
- return `ok=false` from `QueryEmbeddingCacheKey` when one request should bypass caching entirely

`UpsertDocument` is optimized for steady-state writes:

- if content and retrieval-visible metadata are unchanged, it returns without rewriting document or chunk rows
- if only `title`, `source_uri`, `tags`, or `metadata` changed, it updates those fields in place without re-splitting or re-embedding

`UpsertDocument` does not try to guess whether your splitter or embedder recipe changed.
If you roll out a new splitter or embedder and need stored chunks or embeddings rebuilt for unchanged content, call `ReindexDocument`.

Example:

```go
_, err = store.ReindexDocument(ctx, simplykb.UpsertDocumentRequest{
    DocumentID: "doc-1",
    Title:      "BM25 notes",
    Content:    "BM25 is still strong for exact names, short queries, and logs.",
})
if err != nil {
    panic(err)
}
```

If you need diagnostics for one search call without changing ranking behavior, use `SearchDetailed`:

```go
response, err := store.SearchDetailed(ctx, simplykb.SearchRequest{
    Query: "exact names and logs",
    Limit: 5,
})
if err != nil {
    panic(err)
}

fmt.Println("hits", len(response.Hits))
fmt.Println("mode", response.Diagnostics.Mode)
fmt.Println("vector candidates", response.Diagnostics.VectorCandidateCount)
fmt.Println("query cache status", response.Diagnostics.QueryEmbeddingCacheStatus)
fmt.Println("query cache hit", response.Diagnostics.QueryEmbeddingCacheHit)
```

If you enable `QueryEmbeddingCacheSize`, `New` requires your embedder to implement `QueryEmbeddingCacheKeyer`.
That makes cache eligibility explicit instead of silently falling back to re-embedding.

`QueryEmbeddingCacheHit` is a convenience boolean for the hot-path question.
If you need the full diagnosis, read `QueryEmbeddingCacheStatus`:

- `disabled`
- `bypassed`
- `miss`
- `hit`
- `not_applicable`

That lets the embedder decide whether:

- normalized query text alone is enough
- request context must be folded into the cache key
- one request should bypass caching entirely

Example:

```go
func (e *TenantAwareEmbedder) QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, error) {
    tenant, _ := ctx.Value(tenantContextKey{}).(string)
    if tenant == "" {
        return "", false, nil
    }
    return tenant + ":" + normalizedQuery, true, nil
}
```

## Operator Checks

When setup, migration, or retrieval behavior looks suspicious, start with:

```bash
make doctor
```

`make doctor` inspects the database URL you configured. It does not automatically start ParadeDB for you.

This command prints:

- the redacted database URL it connected to
- current database, schema, and search path
- whether `pg_search` and `vector` support are available
- whether the migrations table exists, which versions are applied, and what the current expected version is
- the current embedding column type
- current document and chunk counts

If you changed the local ParadeDB port:

```bash
PARADEDB_PORT=45432 make db-up
PARADEDB_PORT=45432 make doctor
```

## Run Tests

```bash
go test ./...
make integration-test
make verify
```

Use the integration command whenever a change affects setup, migrations, document normalization, or retrieval behavior.

Integration coverage now includes:

- legacy schema upgrade regression
- a fresh external Go module consumer check
- a provider-shaped OpenAI-compatible embedder path against a real ParadeDB database

`make verify` is the repo-level acceptance command. It covers unit tests, `go vet`, the quickstart smoke path, `make doctor`, and integration tests.

Use `make verify` before release work or any public-facing setup change.

## Benchmarks

Capacity notes in this README are design targets, not universal throughput promises.

Use these commands when you want a reproducible local baseline:

```bash
make benchmark
make integration-benchmark
```

`make benchmark` exercises the built-in hash embedder and default splitter.

`make integration-benchmark` exercises real ParadeDB-backed upsert and hybrid search against the local database.

Treat the output as a comparison baseline across commits or machines, not as a guaranteed production SLA.

## Compatibility

The current documented development target is:

| Component | Expected baseline |
| --- | --- |
| Go | `1.25.x` from [go.mod](go.mod) |
| Database | ParadeDB via the bundled [docker-compose.yml](docker-compose.yml) |
| Docker | Docker Engine or Docker Desktop with Compose support |
| OS assumptions | macOS and Linux style shell workflow |

If you want the most predictable local setup, use the bundled Compose file instead of wiring your own database first.

## Documentation Map

If you want the current documentation tree in one place, start with [docs/README.md](docs/README.md).

## Troubleshooting

Common setup and runtime issues are documented in [docs/troubleshooting.md](docs/troubleshooting.md).

The most common ones are:

- database not reachable
- using plain Postgres instead of ParadeDB
- embedding dimension mismatch after changing config
- uncertainty about current schema, extension support, or migration state
- searches returning no hits because data was never indexed into the expected collection
- integration tests being skipped because `SIMPLYKB_DATABASE_URL` is not set

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup, test expectations, and scope guardrails.

## Support

Before opening a new issue:

- check [docs/troubleshooting.md](docs/troubleshooting.md) for common setup and runtime fixes
- use the bug report or feature request template so the issue has the right reproduction and scope details
- treat platform-style expansion requests as design discussions, not default roadmap items

If you are reporting a regression in setup, migrations, or retrieval behavior, include the exact verification you ran.

## Data Envelope

This is the current design target for a single-node production deployment, not a benchmark claim:

- single ParadeDB node
- one embedding model per deployment
- up to 100 collections
- up to 100,000 documents
- up to 1,000,000 chunks
- chunk size around 800 to 1,000 characters
- default candidate fan-out 20 per branch

If you need multi-tenant auth, distributed writes, or much larger scale, that should be a deliberate next system, not an accidental extension of this SDK.
