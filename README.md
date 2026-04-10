# simplykb

[![CI](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml/badge.svg)](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml)

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

- PDF parsing, OCR, or a document ingestion platform
- multi-tenant auth or ACL logic inside the SDK
- distributed writes or a large async indexing pipeline
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
- document upsert and delete
- default chunk splitter
- stable chunk ids
- hybrid recall with reciprocal rank fusion
- local Docker Compose for ParadeDB
- integration tests
- quickstart example

Not supported yet:

- PDF parsing
- async ingestion pipeline
- ACL and multi-tenant auth
- reranker
- dashboard
- distributed write path

Choose another system if you need those capabilities today.

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

If port `25432` is already in use on your machine, override the local ParadeDB port:

```bash
PARADEDB_PORT=35432 make smoke
```

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
PARADEDB_PORT=35432 make db-up
PARADEDB_PORT=35432 go run ./examples/quickstart
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

## Production Notes

`HashEmbedder` exists for tests, local demos, and zero-cost smoke verification.
Do not treat it as a production embedding strategy.

Production deployments should:

- plug in a real provider through the `Embedder` interface
- keep one embedding dimension per deployment
- avoid changing embedding dimensions against an already-migrated schema
- treat `simplykb` as a narrow embedded SDK, not a platform boundary

## Run Tests

```bash
go test ./...
make integration-test
```

Use the integration command whenever a change affects setup, migrations, document normalization, or retrieval behavior.

## Compatibility

The current documented development target is:

| Component | Expected baseline |
| --- | --- |
| Go | `1.25.x` from [go.mod](go.mod) |
| Database | ParadeDB via the bundled [docker-compose.yml](docker-compose.yml) |
| Docker | Docker Engine or Docker Desktop with Compose support |
| OS assumptions | macOS and Linux style shell workflow |

If you want the most predictable local setup, use the bundled Compose file instead of wiring your own database first.

## Troubleshooting

Common setup and runtime issues are documented in [docs/troubleshooting.md](docs/troubleshooting.md).

The most common ones are:

- database not reachable
- using plain Postgres instead of ParadeDB
- embedding dimension mismatch after changing config
- searches returning no hits because data was never indexed into the expected collection
- integration tests being skipped because `SIMPLYKB_DATABASE_URL` is not set

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup, test expectations, and scope guardrails.

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
