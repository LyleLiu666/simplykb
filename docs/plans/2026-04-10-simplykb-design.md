# simplykb Design

## Goal

Build a short, low-entropy, production-grade knowledge base SDK for Go projects.

The project should:

- run locally with `docker compose up`
- use ParadeDB plus `pgvector`
- focus only on text ingestion, chunking, indexing, and recall
- expose a clean embedded SDK instead of a standalone service
- keep the public API small and stable
- stay usable without paid model providers for tests and smoke runs

## Non-goals

- PDF parsing or OCR
- workflow engine, queue, or worker system
- admin console
- complex query planner
- reranker
- multi-database portability

## Chosen architecture

The production target is intentionally narrow: one durable system only, ParadeDB.

The Go application uses an embedded SDK and talks to Postgres directly.
Documents are written synchronously:

1. validate request
2. split text into chunks
3. generate embeddings
4. upsert one document row
5. replace the document's chunk rows

Read path:

1. run BM25 against ParadeDB
2. run vector similarity against `pgvector`
3. fuse the two candidate sets with reciprocal rank fusion
4. return chunk-level hits with score breakdown

This is slower than a fully specialized search platform at high scale, but much lower entropy and operational risk for an SDK-style project.

## Public SDK surface

The SDK exposes four core operations:

- `Migrate`
- `UpsertDocument`
- `DeleteDocument`
- `Search`

The caller also injects an `Embedder`.
That keeps vendor choice outside the core package.

The project ships a built-in `HashEmbedder` only for tests and local smoke runs.
Production users are expected to inject a real embedding provider through the interface.

## Storage model

Two core tables are enough for the production slim SDK:

- `kb_documents`: document-level metadata and hash
- `kb_chunks`: chunk-level recall records

Important design choices:

- one `collection` string acts as the logical namespace
- one embedding dimension per deployment
- one migration table tracks schema versions
- one advisory lock serializes migrations across instances
- updates delete and rebuild all chunks for one document
- no background indexing queue

This gives a very clear contract at the cost of heavier document updates, which is acceptable for the targeted scale envelope.

## Recall strategy

The recall strategy is intentionally simple:

- BM25 over `search_text`
- cosine distance over `embedding`
- reciprocal rank fusion with a fixed `k`

The point is not to be clever.
The point is to be predictable, easy to reason about, and safe to keep in production for single-node workloads.

## Data envelope

The numbers below are design targets, not benchmark results:

- one ParadeDB node
- 100 collections or fewer
- 100,000 documents or fewer
- 1,000,000 chunks or fewer
- 256 to 1024 embedding dimensions
- per-document text payload up to 2 MB
- chunk size around 800 to 1,000 characters with 120 characters overlap

If real workloads exceed this, the project should split into explicit ingestion and recall subsystems instead of accumulating shortcuts.

## Extensibility

The next safe extension points are:

- pluggable metadata filters
- optional reranker
- custom splitter implementations
- external embedding providers
- async reindexing for large documents

The next unsafe extension points, which should be resisted until needed:

- exposing raw SQL or raw ParadeDB query contracts
- mixing tenant auth into the SDK core
- adding many parallel search versions
- introducing worker queues without a durable state model
