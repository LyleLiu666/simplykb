# simplykb Hardening Design

## Document Purpose

This document turns a useful external feedback note into a concrete hardening plan for `simplykb`.

It is not a point-by-point defense of the current repository, and it is not a promise to grow `simplykb` into a broader search platform.

The goal is narrower:

1. keep the parts of the feedback that are factually correct and strategically useful
2. separate real improvement targets from deliberate product boundaries
3. define the next changes in a way that keeps the SDK thoughtful, well-crafted, and engineered

## North Star

`simplykb` should remain a narrow embedded Go SDK that gives a service hybrid recall with one ParadeDB-backed database and a small, predictable API.

Any change in this document should be judged against that standard.
If a change makes `simplykb` feel like a platform product, the change is suspect.

## What We Accept From The Feedback

The following observations are accepted as directionally correct for the current codebase:

### 1. Write amplification is real on document updates

Today, [`UpsertDocument`](../../store.go) always splits content, generates embeddings, deletes old chunks, and reinserts the full chunk set.

The current implementation stores `content_hash`, but it does not use that value to avoid unnecessary work.
That means repeated writes of unchanged content do more CPU, network, and database work than necessary.

### 2. The search path is intentionally simple, but still early

Today, [`Search`](../../store.go) embeds the query synchronously on every vector or hybrid call and fuses keyword plus vector candidates with fixed reciprocal rank fusion.

That simplicity is aligned with the current product shape, but the current path does not yet expose retrieval diagnostics, repeated-query optimization, or a clean extension point for higher-cost ranking stages.

### 3. The public query surface is narrow

Today, [`SearchRequest`](../../types.go) intentionally exposes a small set of fields and both retrieval branches use the same simple metadata containment filter.

That is acceptable for the current SDK shape, but it also means more advanced policy, tenancy, weighting, and paging behavior is not yet represented in the public contract.

### 4. The SDK is still early in maturity

The repository already shows healthy habits: tests, integration coverage, diagnostics, and release discipline.
Even so, `v0.1.1` should still be treated as an early SDK, not as a battle-worn dependency that has absorbed years of production edge cases.

## What We Do Not Accept As Defects

The following points are important context, but they should not be treated as bugs to "fix":

### 1. ParadeDB plus `pgvector` is a design choice, not an accidental lock-in

The repository explicitly states that `simplykb` targets one ParadeDB-backed database and is not trying to be plain-Postgres portable.

Portability beyond that boundary is a separate product decision.
It should not be smuggled into this hardening plan as if the current design simply forgot about it.

### 2. ACL and multi-tenant policy should not be rushed into the SDK core

The current repository already says that ACL and multi-tenant auth logic are out of scope.

That boundary should remain unless there is clear evidence that a narrow embedded SDK can no longer serve its intended workloads without owning that policy layer.

## Design Goals

This hardening plan has four goals:

1. eliminate avoidable write work without changing the external `UpsertDocument` contract
2. improve search introspection and repeated-query efficiency without bloating the default search path
3. preserve the narrow SDK boundary instead of chasing every downstream feature request
4. raise operational confidence through tests, diagnostics, and benchmarks before broadening capability

## Non-goals

This document does not propose:

- plain Postgres compatibility
- a background ingestion platform
- ACL or tenant-policy enforcement inside the SDK core
- a hosted search service shape
- an immediate pagination or boosting framework bolted onto the current search contract
- a required reranker in the default path

## Chosen Design

## 1. Classify Upserts Before Reindexing

The first hardening change should be on the write path because it addresses a real cost today without expanding the product boundary.

`UpsertDocument` should move from a single "always rebuild" behavior to a three-way decision when the indexing recipe is unchanged:

| Current text | Chunk-projected metadata | Action |
| --- | --- | --- |
| unchanged | unchanged | no-op |
| unchanged | changed | metadata-only refresh |
| changed | any | full reindex |

For this decision, "chunk-projected metadata" means every field duplicated onto chunk rows and therefore visible to retrieval:

- `title`
- `source_uri`
- `tags`
- `metadata`

This distinction matters.
A naive "same content hash means skip everything" shortcut would be wrong, because chunk rows would keep stale metadata after a title or metadata-only update.

### Indexing recipe rule

The current `Embedder` and `Splitter` interfaces do not expose a stable identity or revision.
That means the SDK cannot reliably infer when unchanged content still needs reindexing because the chunking or embedding pipeline changed.

So the hardening plan must make this explicit:

- automatic `noop` or `metadata_refresh` is only valid while the caller intends to keep the same indexing recipe
- splitter or embedder rollouts require an explicit caller-driven full rebuild
- the SDK should not pretend it can infer that rollout safely from the current interfaces

The first phase should therefore add a separate explicit API for forced rebuilds:

```go
func (s *Store) ReindexDocument(ctx context.Context, req UpsertDocumentRequest) (DocumentStats, error)
```

Rules:

- `UpsertDocument` may choose `noop`, `metadata_refresh`, or `reindex`
- `ReindexDocument` always performs a full reindex
- operational rollout of splitter or embedder changes must use `ReindexDocument` when documents need to be refreshed without content changes

This gives the SDK a truthful degradation path without requiring the first hardening phase to invent schema state it cannot prove.

### Write-path algorithm

The write path should evolve like this:

1. normalize and validate the request
2. compute the incoming `content_hash`
3. read the existing document snapshot
4. if the caller chose `ReindexDocument`, force `reindex`
5. otherwise classify the write as `noop`, `metadata_refresh`, or `reindex`
6. only split content and generate embeddings for the `reindex` path
7. persist the chosen path transactionally

### Concurrency rule

The implementation should not hold a row lock across splitter or embedder work.
That would turn expensive external work into a lock-duration amplifier.

Instead:

1. read the current snapshot outside the expensive work to choose the likely path
2. if the likely path is `noop` or `metadata_refresh`, re-check the row inside a transaction with row locking before committing
3. if the likely path is `reindex`, allow rare redundant computation rather than long document-row lock times

This keeps correctness while favoring low contention over perfect precomputation efficiency.

### Reclassification outcome

The locked re-check must have an explicit outcome when the cheap pre-read is no longer valid.

Rules:

- if the locked snapshot still matches `noop`, commit `noop`
- if the locked snapshot still matches `metadata_refresh`, commit `metadata_refresh`
- if the locked snapshot now requires `reindex`, abort the cheap-path transaction before any row rewrite and restart once as a full reindex path
- the SDK should not widen lock scope to cover splitter or embedder work just to preserve the first classification

To avoid retry loops, the internal restart should happen at most once per call.
If the restarted call still cannot complete because concurrent writers keep moving the target, the SDK should return a typed concurrency error and let the caller retry.

Suggested shape:

```go
var ErrDocumentChangedConcurrently = errors.New("document changed concurrently")
```

Returned errors for this case should wrap `ErrDocumentChangedConcurrently` so callers can use `errors.Is`.

### Write visibility

The first hardening phase should not grow `DocumentStats`.

That keeps the existing return shape stable while the write-path behavior changes underneath it.
For phase 1, visibility should come from:

- integration tests that prove which path ran
- benchmarks that show repeated-upsert cost changes
- optional future follow-up work, if needed, through a new opt-in detailed response instead of mutating `DocumentStats`

### Failure semantics

The write path should keep the current transactional guarantee.

- `noop` returns successfully without rewriting rows
- `metadata_refresh` updates document and chunk rows in one transaction
- `reindex` keeps the existing all-or-nothing behavior

If the SDK cannot complete the chosen write path, it should return an error and leave persisted state unchanged.
If a caller needs a guaranteed full rebuild despite unchanged content, it should use `ReindexDocument` instead of hoping the SDK infers that need.

### Timestamp rule

`updated_at` should only move when persisted document or chunk state actually changes.
A true no-op should not silently rewrite timestamps.

If a caller wants heartbeat-like "touch" semantics, that should stay in the business layer.

## 2. Harden Search Without Breaking The Simple Default

The second hardening area is search.
The current path is simple on purpose, so the plan is to preserve the default behavior and add small, opt-in extensions around it.

### Keep the current default search shape

The existing `Search(ctx, req)` contract should stay intact as the lowest-entropy path.

That means:

- keyword-only, vector-only, and hybrid modes stay supported
- reciprocal rank fusion remains the default merge strategy
- the current simple request shape remains the primary entrypoint

### Add a response type for opt-in diagnostics

Instead of overloading `SearchHit` or adding many ad hoc request fields, the SDK should introduce a second response-oriented entrypoint for callers who need more insight.

Suggested shape:

- `SearchDetailed(ctx, req) (SearchResponse, error)`
- existing `Search(ctx, req)` can delegate to `SearchDetailed` and return `response.Hits`

Suggested additive response types:

```go
type SearchResponse struct {
    Hits         []SearchHit
    Diagnostics  SearchDiagnostics
}
```

```go
type SearchDiagnostics struct {
    Mode                   SearchMode
    TotalDuration          time.Duration
    KeywordDuration        time.Duration
    VectorDuration         time.Duration
    KeywordCandidateCount  int
    VectorCandidateCount   int
    FusedCandidateCount    int
    QueryEmbeddingCacheHit bool
    HadContextDeadline     bool
}
```

`SearchDiagnostics` should focus on facts the SDK already knows or can measure cheaply:

- total duration
- keyword branch duration
- vector branch duration
- keyword candidate count
- vector candidate count
- fused hit count before truncation
- whether the query embedding came from cache
- whether the caller provided a deadline on `ctx`

`SearchDetailed` should always populate diagnostics.
The simpler `Search` wrapper should keep returning only hits.

Response invariants:

- `Search(ctx, req)` and `SearchDetailed(ctx, req)` must return the same `Hits` ordering
- diagnostics must describe what happened, not alter ranking
- unused branch fields stay at zero values in keyword-only or vector-only searches

### Search failure semantics

The detailed path should not invent partial-success semantics in the first version.

- if keyword search fails, the call fails
- if vector embedding or vector search fails, the call fails
- if the optional cache is unavailable or bypassed, search falls back to direct embedding and still succeeds when the embedder succeeds

This keeps `Search` and `SearchDetailed` behavior aligned and avoids a half-degraded result contract that callers have to guess about.

### Respect context instead of inventing duplicate timeout knobs

The current search path already accepts `context.Context`, and integration coverage shows that canceled contexts propagate correctly.

That means the first hardening step should not be to add a separate timeout field to `SearchRequest`.
The caller already owns the top-level budget through `ctx`.

If stage-level budgeting is needed later, it should derive from the remaining context deadline instead of introducing an unrelated second timeout system.

### Add an optional in-process query embedding cache

Repeated query embedding is a real cost for hybrid or vector traffic.
The first optimization should be a small in-process cache, disabled by default.

Important constraint:
the current `Embedder` interface does not expose a stable model identity.
Because of that, the cache must be scoped to a single `Store` instance and should never be shared globally across stores.

### Cache contract

The first cache version must only be treated as correct for context-invariant embedders.

That means:

- embedding output is determined by normalized query text plus store-level configuration
- `context.Context` is used only for cancellation, deadlines, and transport-scoped concerns
- the embedder does not switch model, tenant, locale, or routing behavior based on request-scoped context values

If an embedder uses context as part of semantic routing, the caller should leave `QueryEmbeddingCacheSize` at `0`.
The first version should not attempt to inspect or hash arbitrary context values.

Safe cache rules:

- nil or zero-sized cache means current behavior
- cache key is based on the normalized query text within one `Store`
- cache entries are invalidated by store replacement, not by global state
- a cache miss or cache failure falls back to direct embedding
- cache access must be safe under concurrent `Search` and `SearchDetailed` calls from many goroutines
- the first version does not need singleflight-style coalescing; duplicate cold-miss embeddings are acceptable, but races and corrupted cache state are not

This keeps the optimization bounded and avoids pretending the SDK can safely deduplicate embeddings across unknown embedder implementations.

### Minimal config surface

The first cache version should add only one public configuration knob:

- `Config.QueryEmbeddingCacheSize int`

Rules:

- default `0` means disabled
- positive values enable a bounded per-store cache
- cache implementation should be a simple bounded in-memory LRU, not a distributed or cross-process cache
- no separate TTL knob in the first version
- no separate diagnostics knob in the first version

This keeps the public surface small and gives the feature an obvious off switch.

### SDK compatibility stance

`simplykb` is still a pre-1.0 SDK, so exported APIs may evolve.
Even so, this hardening plan should prefer less brittle extension shapes:

- prefer adding new methods over mutating existing result structs
- keep `Search` intact and add `SearchDetailed` separately
- call out any exported `Config` growth explicitly in changelog and release notes
- keep examples and documentation on keyed struct literals rather than positional ones

This is especially important because the repository already presents a small stable SDK surface as part of its value proposition.

### Defer reranking until diagnostics justify it

The feedback is fair that the current retrieval path has no reranker.
Even so, reranking should not be the first hardening move.

The safer order is:

1. reduce write waste
2. expose search diagnostics
3. measure real failure cases and ranking quality pressure
4. only then decide whether a reranker interface belongs in the SDK

That ordering keeps the project from importing cost and latency before it has evidence.

## 3. Preserve Narrow Boundaries Deliberately

The third hardening theme is not feature expansion.
It is boundary discipline.

### Keep advanced policy outside the core

ACL, multi-tenant authorization, and business-specific post-filtering may still happen above the SDK.
That should not automatically be read as distrust of the SDK.

In many systems, it is ordinary defense-in-depth.

### Defer pagination and boosting until the response model is broader

Pagination and boosting are valid future needs, but they should not be attached casually to the current chunk-level search contract.

They depend on bigger questions:

- are we paging chunks or documents
- is fusion happening before or after boosting
- what ranking evidence should diagnostics expose

Until those decisions are explicit, the SDK should prefer a small, stable search API over half-finished expansion.

## 4. Raise Maturity Through Verification First

The fourth hardening theme is maturity.
This should be earned through repeatable verification, not through broader claims.

### Phase 1 acceptance criteria

For the write-path hardening work:

- unchanged content plus unchanged metadata returns `noop`
- unchanged content plus changed metadata updates document and chunk metadata without re-embedding
- changed content still performs full reindex
- `ReindexDocument` forces a full rebuild even when content and metadata are unchanged
- integration tests prove unchanged documents do not call the embedder again
- integration tests prove splitter or embedder rollout can be handled by explicit forced reindex
- benchmarks show reduced repeated-upsert cost for unchanged documents

### Phase 2 acceptance criteria

For the search hardening work:

- `Search` remains source-compatible
- `SearchDetailed` returns stable diagnostics without changing ranking behavior
- a repeated query can hit the optional cache inside one `Store`
- canceled or expired contexts still fail cleanly
- integration tests cover cache disabled, cache enabled, and deadline propagation
- parallel tests cover concurrent cache access and repeated cold-key queries

### Documentation and operator criteria

Each phase should update:

- [`README.md`](../../README.md) for public capability wording
- [`docs/troubleshooting.md`](../troubleshooting.md) for new operator-visible behavior
- benchmarks or benchmark notes when a change is meant to reduce cost
- release notes when exported APIs or rollout requirements change

Operator-facing acceptance for phase 1:

- public docs must state that unchanged documents do not automatically refresh after splitter or embedder changes
- public docs must show `ReindexDocument` as the required rebuild path for indexing-recipe rollouts
- troubleshooting guidance must include recipe-change rollout as a first-class case, not a footnote

## Rollback Plan

Rollback should stay simple.

### Phase 1 rollback

If write-path classification causes unexpected behavior, the SDK should be able to fall back to the current always-reindex path without a schema migration.

If `UpsertDocument` classification proves unsafe in practice, callers can temporarily route all refreshes through `ReindexDocument` while the optimization is corrected.

### Phase 2 rollback

If the cache or diagnostics path causes instability:

- keep `Search` as the stable compatibility path
- set `QueryEmbeddingCacheSize` to `0` to disable caching
- let callers stop using `SearchDetailed` without affecting existing integrations

## Knowledge Anchoring Plan

The non-obvious decisions in this document should not live only in prose.
They should be anchored in code and tests:

- no-op versus metadata-refresh versus reindex behavior should be locked by integration tests
- explicit forced-reindex behavior should be locked by integration tests
- cheap-path reclassification to full reindex should be locked by integration or concurrency tests
- retryable concurrent-change behavior should be anchored by a typed error plus tests using `errors.Is`
- `Search` and `SearchDetailed` ranking equivalence should be locked by tests
- cache-disabled-by-default behavior should be enforced in `Config.normalized()` and `Config.validate()`
- cache use must be documented as valid only for context-invariant embedders
- concurrent cache access should be covered by race-oriented tests

## Expected File Areas

The likely implementation footprint for this plan is:

- [`store.go`](../../store.go)
- [`types.go`](../../types.go)
- [`config.go`](../../config.go)
- [`integration_test.go`](../../integration_test.go)
- [`benchmark_test.go`](../../benchmark_test.go)
- [`README.md`](../../README.md)
- [`docs/troubleshooting.md`](../troubleshooting.md)

The initial implementation should avoid schema churn unless a concrete gap appears during implementation.
The current design can likely ship the first phase with logic changes, one additive method, and tests only.

## Alternatives Considered

### 1. Only add a content-hash short-circuit

Rejected as incomplete.

That would save work for exact content repeats, but it would also miss metadata-only updates because chunk rows currently duplicate retrieval-visible metadata.

### 2. Expand `SearchRequest` into a large advanced query object immediately

Rejected for now.

That would add entropy faster than the current repository has evidence for.
The safer move is to keep `Search` simple and add a second, more informative response path when needed.

### 3. Generalize the database target beyond ParadeDB

Rejected as out of scope.

That is a product repositioning question, not a hardening patch.

### 4. Persist an indexing revision in schema in the first hardening phase

Deferred for now.

That approach could eventually support automatic detection of splitter or embedder rollouts, but it also adds schema churn and a broader migration story before the repository has proved the simpler explicit `ReindexDocument` path is insufficient.

## Rollout Order

The intended rollout order is:

1. explicit `ReindexDocument` path for forced rebuilds
2. write-path classification and no-op avoidance
3. metadata-only refresh path
4. `SearchDetailed` plus diagnostics
5. optional in-process query embedding cache
6. re-evaluate reranker, boosting, and pagination only after real usage evidence

## Final Recommendation

The external feedback is most useful when translated into this narrower conclusion:

- fix avoidable write amplification now
- improve retrieval observability next
- keep advanced retrieval features evidence-driven
- protect the SDK's intentionally narrow boundary

That path keeps the current strengths of `simplykb` intact while addressing the most justified weaknesses first.
