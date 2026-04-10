# simplykb

`simplykb` is a slim production-oriented Go knowledge base SDK for text-first recall workloads.

It is designed to stay short without being toy-like:

- one ParadeDB instance
- one Go SDK
- versioned in-process migrations
- synchronous and deterministic document upsert
- built-in text chunking
- BM25 plus vector recall
- no queue, no worker, no extra search service

## Why this shape

This project is optimized for low entropy:

- Postgres remains the only durable system
- ParadeDB gives BM25 inside Postgres
- `pgvector` gives vector similarity in the same database
- Go projects embed the SDK directly instead of calling another HTTP service

## Project status

This repository targets a narrow but production-usable shape:

- single database
- single embedding model per deployment
- embedded SDK, not platform product
- predictable schema and migration path
- explicit limits instead of hidden magic

What is already included:

- versioned schema migration with advisory lock
- document upsert and delete
- default chunk splitter
- stable chunk ids
- built-in local hash embedder for tests and local smoke runs
- hybrid recall with reciprocal rank fusion
- Docker Compose for local ParadeDB
- quickstart example

What is intentionally not included yet:

- PDF parsing
- async ingestion pipeline
- ACL and multi-tenant auth
- reranker
- dashboard
- distributed write path

## Quick start

Start ParadeDB:

```bash
docker compose up -d
```

Run the example:

```bash
SIMPLYKB_DATABASE_URL=postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable \
go run ./examples/quickstart
```

Run tests:

```bash
go test ./...
SIMPLYKB_DATABASE_URL=postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable go test ./... -run Integration
```

## Public API

```go
store, err := simplykb.New(ctx, simplykb.Config{
    DatabaseURL:         os.Getenv("SIMPLYKB_DATABASE_URL"),
    DefaultCollection:   "docs",
    EmbeddingDimensions: 256,
    MaxConns:            8,
    Embedder:            simplykb.NewHashEmbedder(256),
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

`HashEmbedder` is only for tests, local demos, and zero-cost smoke verification.
Real production usage should plug in your own embedding provider through the `Embedder` interface.

## Data envelope

This is the current design target for a single-node production deployment, not a benchmark claim:

- single ParadeDB node
- one embedding model per deployment
- up to 100 collections
- up to 100,000 documents
- up to 1,000,000 chunks
- chunk size around 800 to 1,000 characters
- default candidate fan-out 20 per branch

If you need multi-tenant auth, distributed writes, or much larger scale, that should be a deliberate next system, not an accidental extension of this SDK.
