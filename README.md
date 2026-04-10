# simplykb

[![CI](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml/badge.svg)](https://github.com/LyleLiu666/simplykb/actions/workflows/ci.yml)

`simplykb` is a slim production-oriented Go knowledge base SDK for text-first recall workloads.

It is built for Go services that want BM25 plus vector recall inside Postgres, without standing up a separate search service, queue, or worker fleet.

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

## Why This Shape

This project is optimized for low entropy:

- Postgres remains the only durable system
- ParadeDB gives BM25 inside Postgres
- `pgvector` gives vector similarity in the same database
- Go projects embed the SDK directly instead of calling another HTTP service

The goal is not to be the biggest knowledge base system.
The goal is to be the clearest small one.

## Current Fit

This repository targets a narrow but production-usable shape:

- single database
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

## Quick Start

### 1. Fastest local smoke run

```bash
make smoke
```

This command:

- starts the bundled ParadeDB database
- waits until the database health check passes
- runs the quickstart example against the default local URL

### 2. Manual path

If you want to see each step explicitly:

```bash
make db-up
make db-status
go run ./examples/quickstart
```

The quickstart example already defaults to:

```bash
postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable
```

Set `SIMPLYKB_DATABASE_URL` only if you changed the local port, credentials, or database name.

If `25432` is already in use on your machine, choose another port when starting the bundled database:

```bash
PARADEDB_PORT=35432 make db-up
SIMPLYKB_DATABASE_URL=postgres://simplykb:simplykb@localhost:35432/simplykb?sslmode=disable \
go run ./examples/quickstart
```

This repository expects ParadeDB, not plain Postgres.

### 3. Run tests

Example output shape:

```text
indexed doc-bm25 with 1 chunks
indexed doc-vector with 1 chunks
indexed doc-hybrid with 1 chunks

Top hits:
- doc-bm25 chunk=0 score=...
  snippet: ...
```

Success signals:

- you see three `indexed doc-...` lines
- you see a `Top hits:` section
- at least one search hit is returned
- you do not need exact scores or snippet text to match byte-for-byte

```bash
go test ./...
make integration-test
```

Use the integration command whenever a change affects setup, migrations, document normalization, or retrieval behavior.

## Public API

Demo API example:

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

## Production Notes

`HashEmbedder` exists for tests, local demos, and zero-cost smoke verification.
Do not treat it as a production embedding strategy.

Production deployments should:

- plug in a real provider through the `Embedder` interface
- keep one embedding dimension per deployment
- avoid changing embedding dimensions against an already-migrated schema
- treat `simplykb` as a narrow embedded SDK, not a platform boundary

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
