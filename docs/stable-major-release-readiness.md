# Stable Major Release Readiness

Status: active release guidance for a future stable major release such as `v1.0.0`.
Audience: maintainers preparing a release decision.

`RELEASING.md` is the canonical process for every release.
This document adds the extra bar for a stable major release, where the project promises a stable public API and low-surprise behavior.

Historical setup work for this document is archived at [plans/archive/2026-04-24-stable-major-release-readiness-handoff.md](./plans/archive/2026-04-24-stable-major-release-readiness-handoff.md).
One-time validation records live under [release-validation/README.md](./release-validation/README.md).

## Current Baseline

Verified against the repository on 2026-04-28:

| Area | Current evidence |
| --- | --- |
| Latest local tag | `v0.3.0` from `git tag --sort=-creatordate` |
| Module path | `github.com/LyleLiu666/simplykb` in [go.mod](../go.mod) |
| Go baseline | `1.25.0` in [go.mod](../go.mod) |
| Database baseline | ParadeDB image pinned in [docker-compose.yml](../docker-compose.yml) and integration CI |
| Release process | [RELEASING.md](../RELEASING.md) |
| Product contract | Embedded Go SDK, one ParadeDB database, no platform subsystem, as stated in [README.md](../README.md) |

This baseline does not mean `v1.0.0` is approved.
It only names the current repo state that the stable release decision should start from.

## Stable Release Promise

Publishing `v1.0.0` means the project is promising:

- exported Go APIs remain compatible until the next major version
- documented setup, migration, upsert, reindex, delete, and search behavior does not change silently
- new fields or options are optional and have safe defaults
- schema migrations protect users already on published releases
- the SDK stays embedded in the caller's Go service instead of becoming a separate platform

Stable does not mean feature complete.
For `simplykb`, stability is about a small API, predictable operations, and honest release evidence.

## Public Surface To Freeze

Review this surface before tagging `v1.0.0`:

| Category | Public surface | Source |
| --- | --- | --- |
| Construction | `New(ctx, Config)` | [store.go](../store.go), [config.go](../config.go) |
| Store lifecycle | `Close`, `Ping`, `Migrate`, `EnsureSchema` | [store.go](../store.go) |
| Document writes | `UpsertDocument`, `ReindexDocument`, `DeleteDocument` | [store.go](../store.go) |
| Search | `Search`, `SearchDetailed` | [store.go](../store.go), [types.go](../types.go) |
| Configuration | `Config` fields and defaults | [config.go](../config.go) |
| Extension points | `Embedder`, `QueryEmbeddingCacheKeyer`, `Splitter` | [types.go](../types.go) |
| Request and response types | `UpsertDocumentRequest`, `DocumentStats`, `SearchRequest`, `SearchHit`, `SearchResponse`, `SearchDiagnostics`, `ChunkDraft` | [types.go](../types.go) |
| Built-in helpers | `NewDefaultSplitter`, `DefaultSplitter`, `NewHashEmbedder`, `HashEmbedder` | [splitter.go](../splitter.go), [embedder_hash.go](../embedder_hash.go) |
| Search constants | `SearchMode*`, `QueryEmbeddingCacheStatus*` | [types.go](../types.go) |

If any item above needs a breaking change, make that change before `v1.0.0`.
After `v1.0.0`, breaking changes require a later major version.

## Required Gates

Clear every gate before publishing `v1.0.0`:

| Gate | Required evidence | Minimum verification |
| --- | --- | --- |
| API freeze reviewed | Maintainer confirms the public surface above is acceptable as the stable baseline | `rg -n "func New|func \\(.*\\) (Close|Ping|Migrate|EnsureSchema|UpsertDocument|ReindexDocument|DeleteDocument|Search|SearchDetailed)|type (Config|Embedder|QueryEmbeddingCacheKeyer|Splitter|UpsertDocumentRequest|DocumentStats|SearchRequest|SearchHit|SearchResponse|SearchDiagnostics|ChunkDraft)" -g '*.go'` |
| Migration reliability | Released schema states upgrade cleanly | `make integration-test` |
| Local developer path | Quickstart, diagnostics, unit tests, vet, and integration tests pass | `make verify` |
| External module usage | A fresh module can fetch the candidate tag, import the package, and compile against the public surface | Commands in the external module check below |
| Real consumer feedback | At least one real or representative Go service exercises migrate, upsert, reindex, search, and delete | Evidence note under `docs/release-validation/` |
| Benchmark comparison | Schema, retrieval, or setup changes have a recorded local comparison | `make integration-benchmark` when those areas changed |
| Product boundary | Review confirms the diff did not add PDF/OCR ingestion, ACL policy, dashboards, hosted service behavior, or async platform behavior to the core SDK | `git diff --name-only` plus reviewer inspection |
| Release notes | Setup, migration, search behavior, examples, docs, and breaking-change impact are called out | `rg -n "setup|migration|search|example|breaking" CHANGELOG.md` |
| Documentation drift | Public docs still match code, config, examples, and Make targets | Review [README.md](../README.md), [docs/troubleshooting.md](./troubleshooting.md), [CONTRIBUTING.md](../CONTRIBUTING.md), and examples |

If a gate fails, do not tag `v1.0.0`.
Fix the blocker, add or update verification, then rerun the relevant gate.

## External Module Check

Run this from a temporary directory outside the repository.
Use the actual candidate tag in place of `<candidate-tag>`.

```bash
tmpdir="$(mktemp -d)"
cd "$tmpdir"
go mod init example.com/simplykb-check
go get github.com/LyleLiu666/simplykb@<candidate-tag>
cat > main.go <<'GO'
package main

import (
	"context"

	simplykb "github.com/LyleLiu666/simplykb"
)

func main() {
	ctx := context.Background()
	store, err := simplykb.New(ctx, simplykb.Config{})
	if err == nil && store != nil {
		defer store.Close()
		_ = store.Ping(ctx)
		_ = store.Migrate(ctx)
		_ = store.EnsureSchema(ctx)
		_, _ = store.UpsertDocument(ctx, simplykb.UpsertDocumentRequest{DocumentID: "doc", Content: "hello"})
		_, _ = store.ReindexDocument(ctx, simplykb.UpsertDocumentRequest{DocumentID: "doc", Content: "hello"})
		_, _ = store.Search(ctx, simplykb.SearchRequest{Query: "hello", Mode: simplykb.SearchModeHybrid, Limit: 1})
		_, _ = store.SearchDetailed(ctx, simplykb.SearchRequest{Query: "hello", Mode: simplykb.SearchModeKeyword, Limit: 1})
		_ = store.DeleteDocument(ctx, "", "doc")
	}

	_ = simplykb.NewDefaultSplitter()
	_ = simplykb.NewHashEmbedder(8)
	_ = simplykb.SearchModeVector
	_ = simplykb.QueryEmbeddingCacheStatusDisabled
	_ = simplykb.DocumentStats{}
	_ = simplykb.SearchHit{}
	_ = simplykb.SearchResponse{}
	_ = simplykb.SearchDiagnostics{}
	_ = simplykb.ChunkDraft{}
}
GO
go mod tidy
go test ./...
```

This is a compile-focused check.
It should not require a live database.

## Recommended Path From `v0.3.0`

1. Keep `v0.3.0` as the latest released baseline unless a newer tag exists when release work starts.
2. Publish one final stable candidate before `v1.0.0` if the API or behavior is still being validated.
3. Use the candidate period only for blocker fixes, documentation drift, and evidence collection.
4. Record release evidence under `docs/release-validation/`.
5. Tag `v1.0.0` only after every required gate passes.

Optional improvements must not block `v1.0.0` unless they reveal a problem with the stable promise.
Examples include more ingestion examples, more provider examples, broader benchmark scenarios, and reranking examples outside the core SDK.
