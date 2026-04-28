# Stable Major Release Readiness Handoff

Status: historical execution handoff from 2026-04-24.
Current stable major release guidance lives in [../../stable-major-release-readiness.md](../../stable-major-release-readiness.md).
This file preserves the implementation package and kickoff prompt that produced that guidance.

## Execution Spec

### 0. Metadata

- Task name: Prepare simplykb for a stable major release decision
- Date: 2026-04-24
- Owner: Future coding agent
- Source inputs:
  - `/Users/liu_y/code/goProject/simplykb/README.md`
  - `/Users/liu_y/code/goProject/simplykb/RELEASING.md`
  - `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md`
  - `/Users/liu_y/code/goProject/simplykb/Makefile`
  - `/Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml`
  - `/Users/liu_y/code/goProject/simplykb/.github/workflows/integration.yml`
  - `/Users/liu_y/code/goProject/simplykb/types.go`
  - `/Users/liu_y/code/goProject/simplykb/store.go`

### 1. Mission and Done Definition

#### 1.1 Mission

Define and implement the repository changes required before `simplykb` can honestly publish a stable major version such as `v1.0.0`.

#### 1.2 Done Criteria

- D-001: A stable major release checklist exists in repository documentation and can be followed without hidden context.
- D-002: The checklist separates must-have release blockers from optional improvements.
- D-003: Every must-have blocker maps to a concrete verification command or manual evidence item.
- D-004: The repository states the public API and behavior compatibility promise for a stable major release.
- D-005: The repository documents the recommended release path from the current `v0.2.1` state to a stable major release candidate.
- D-006: `go test ./...` and `go vet ./...` pass after documentation or code changes.

### 2. Scope

#### 2.1 In Scope

- S-001: Add or update documentation that defines the stable major release bar.
- S-002: Add or update release checklist items in `/Users/liu_y/code/goProject/simplykb/RELEASING.md` if the existing release process lacks stable major release gates.
- S-003: Add verification guidance for unit tests, vet, smoke testing, doctor diagnostics, integration tests, integration benchmarks, and external module checks.
- S-004: Preserve the current product boundary: embedded Go SDK, one ParadeDB database, no platform subsystem.
- S-005: Use the current public API shape as the baseline unless code inspection proves a severe flaw.

#### 2.2 Out of Scope

- OOS-001: Do not add PDF parsing, OCR, dashboard, hosted service behavior, ACL enforcement, or multi-tenant policy logic to the core SDK.
- OOS-002: Do not redesign ParadeDB storage unless a failing test proves a correctness problem.
- OOS-003: Do not change the module path `github.com/LyleLiu666/simplykb`.
- OOS-004: Do not publish a Git tag or GitHub release.
- OOS-005: Do not commit changes unless the user explicitly asks for a commit.

#### 2.3 Non-goals

- NG-001: Do not maximize feature count before a stable major release.
- NG-002: Do not convert the SDK into a separate service.
- NG-003: Do not make compatibility promises that the repository cannot verify.
- NG-004: Do not use keyword matching or brittle trigger words as a substitute for semantic behavior in any newly added logic.

### 3. Constraints and Assumptions

#### 3.1 Hard Constraints

- C-001: Follow the repository instruction to use TDD thinking: identify or add verification before making behavior changes.
- C-002: Use plain language in documentation.
- C-003: Keep public behavior low-surprise and explicitly documented.
- C-004: Treat `v1.0.0` as an API and behavior stability promise.
- C-005: Avoid unrelated refactors.
- C-006: Keep file changes minimal and focused on release readiness.

#### 3.2 Assumptions

- A-001: The current latest tag is `v0.2.1`.
- A-002: The project is already beyond prototype quality because it has core SDK methods, migration support, examples, diagnostics, tests, CI, and release documentation.
- A-003: The safer path is to publish a stable candidate version before `v1.0.0`, using a tag such as `v0.3.0` or `v0.9.0`.
- A-004: `v1.0.0` should be published only after at least one real external consumer or fresh external module validates installation and basic use.
- A-005: Documentation-only changes do not require integration benchmarks unless release guidance or benchmark commands are changed.

### 4. Canonical Terms

| Term | Definition | Allowed aliases | Disallowed aliases |
|---|---|---|---|
| Stable major release | A release such as `v1.0.0` that promises stable public API and low-surprise behavior | `v1.0.0`, stable version | feature-complete platform |
| Stable candidate | A pre-`v1.0.0` release used to collect real usage feedback before the stability promise | `v0.3.0`, `v0.9.0`, release candidate | final stable release |
| Public API | Exported Go types, exported functions, exported methods, documented behavior, schema migration behavior, and retrieval behavior | API surface, public surface | internal helper code |
| Core SDK | The embedded Go library that connects to ParadeDB and provides migration, upsert, delete, reindex, and search behavior | SDK core | hosted product |
| Release blocker | A missing condition that must be resolved before publishing `v1.0.0` | must-have gate | nice-to-have |
| Optional improvement | A useful change that must not block `v1.0.0` if the core promise is already met | follow-up | blocker |

### 5. Requirement List

| ID | Type | Requirement statement | Priority | Rationale |
|---|---|---|---|---|
| R-001 | Functional | Document that `v1.0.0` means stable public API and behavior, not a broad feature expansion. | must | Prevents users and future agents from equating stability with feature volume. |
| R-002 | Functional | Document a release path that recommends one stable candidate release before `v1.0.0`. | must | Reduces risk by collecting real usage feedback before the compatibility promise. |
| R-003 | Functional | Document must-have `v1.0.0` blockers with explicit pass evidence. | must | Makes the release decision executable by a low-capability agent. |
| R-004 | Functional | Document optional improvements separately from blockers. | must | Prevents scope creep from delaying a focused stable release. |
| R-005 | Functional | Document the public API baseline that should be frozen for `v1.0.0`. | must | Gives maintainers a concrete compatibility surface. |
| R-006 | Functional | Document migration reliability expectations for all previously released schema states. | must | Protects existing users from upgrade data risk. |
| R-007 | Functional | Document required verification commands for stable release readiness. | must | Converts release confidence into repeatable evidence. |
| R-008 | Functional | Document external consumer verification before final `v1.0.0` tagging. | must | Confirms the module works outside the repository. |
| R-009 | Functional | Document product boundaries that must remain outside the core SDK. | must | Keeps the project low-entropy and aligned with README. |
| R-010 | Non-functional | Keep the new documentation understandable to a non-expert reader. | must | Matches repository communication expectations. |
| R-011 | Non-functional | Do not change runtime behavior unless a release blocker requires a tested code fix. | must | Keeps this task focused and safe. |
| R-012 | Functional | Update the documentation index if a new stable release readiness document is added. | should | Makes the new document discoverable. |

### 6. Interface and Data Contracts

#### 6.1 Inputs

| Input | Source | Type | Validation | Failure behavior |
|---|---|---|---|---|
| Current release policy | `/Users/liu_y/code/goProject/simplykb/RELEASING.md` | Markdown | File exists and contains versioning policy | Hard fail and report missing file |
| Current project contract | `/Users/liu_y/code/goProject/simplykb/README.md` | Markdown | File exists and states scope boundaries | Hard fail and report missing file |
| Current version history | `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md` and Git tags | Markdown and Git metadata | Current latest tag can be identified | Hard fail and ask user if Git metadata is unavailable |
| Public API baseline | `/Users/liu_y/code/goProject/simplykb/types.go` and `/Users/liu_y/code/goProject/simplykb/store.go` | Go source | Exported API can be inspected with `rg` | Hard fail if files are missing |
| Verification commands | `/Users/liu_y/code/goProject/simplykb/Makefile` | Make targets | Required targets exist or missing targets are documented | Hard fail for missing `test` or `vet`; document missing optional targets |

#### 6.2 Outputs

| Output | Consumer | Type | Success criteria |
|---|---|---|---|
| Stable major release readiness documentation | Maintainers and future coding agents | Markdown | Contains blockers, optional improvements, release path, verification commands, and API promise |
| Updated documentation index | Maintainers and readers | Markdown | New document is linked from `/Users/liu_y/code/goProject/simplykb/docs/README.md` when a new document is created under `docs/` |
| Verification summary | User | Plain text | Lists commands run and pass or fail results |

#### 6.3 State Transitions

| From | Event | Guard | To | Notes |
|---|---|---|---|---|
| `v0.2.1` current state | Stable candidate checklist passes | All must-have candidate checks pass | Stable candidate can be tagged | Recommended candidate tag is `v0.3.0` or `v0.9.0` |
| Stable candidate | Real external consumer validation passes | External module check and at least one real integration report pass | `v1.0.0` can be considered | Do not skip the candidate feedback step |
| Stable candidate | API flaw or migration flaw is found | Failing test or documented reproduction exists | Fix before `v1.0.0` | Add regression coverage before code fix |
| Stable major release | Breaking API or behavior change is required | No compatible path exists | Next major version required | Document breaking change in release notes |

### 7. Implementation Plan

| Step ID | Goal | Files | Change action | Commands | Expected evidence |
|---|---|---|---|---|---|
| I-001 | Inspect current release and project docs | `/Users/liu_y/code/goProject/simplykb/README.md`, `/Users/liu_y/code/goProject/simplykb/RELEASING.md`, `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md` | Read files and confirm current contract | `sed -n '1,220p' README.md`; `sed -n '1,220p' RELEASING.md`; `git tag --sort=-creatordate | sed -n '1,20p'` | Latest release and current stability promise are known |
| I-002 | Create stable readiness document | `/Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md` | Add a new Markdown document with stable release bar, blockers, optional improvements, release path, API baseline, and verification commands | `cat docs/stable-major-release-readiness.md` | Document contains all required sections |
| I-003 | Update documentation index | `/Users/liu_y/code/goProject/simplykb/docs/README.md` | Add the new document to the reading path or supporting document list | `sed -n '1,220p' docs/README.md` | Readers can discover the new document |
| I-004 | Check wording quality | `/Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md` | Search for forbidden placeholders and vague words | `pattern='TB''D|et''c\.|as nee''ded|la''ter|prope''rly|so''me|f''ew|so''on'; rg -n "$pattern" docs/stable-major-release-readiness.md` | Command returns no matches |
| I-005 | Run baseline verification | Repository root | Run basic tests and vet | `go test ./...`; `go vet ./...` | Both commands pass |
| I-006 | Report completion | No file change required | Summarize changed files, verification, and remaining human decision points | `git diff -- docs/stable-major-release-readiness.md docs/README.md` | User can review exact changes |
| I-007 | Verify blocker structure | `/Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md` | Confirm every release blocker has evidence, minimum command or evidence, artifact, and pass condition | `rg -n "^- RB-[0-9]{3}:|  - Evidence:|  - Minimum command|  - Minimum evidence:|  - Artifact:|  - Pass condition:" docs/stable-major-release-readiness.md` | Every `RB-` block has the required structural fields |
| I-008 | Verify external consumer compile path | Fresh temporary directory outside the repository | Create a tiny consumer module that imports `simplykb` and compiles against the public API | Commands in section 13.4 | The consumer module compiles and exits with code 0 |

### 8. Verification Plan

| Test ID | Covers requirement IDs | Type | Command | Pass criteria |
|---|---|---|---|---|
| T-001 | R-001, R-002, R-003, R-004, R-005, R-006, R-007, R-008, R-009 | Structural documentation inspection | `rg -n "^- RB-[0-9]{3}:|  - Evidence:|  - Minimum command|  - Minimum evidence:|  - Artifact:|  - Pass condition:" docs/stable-major-release-readiness.md` | Every `RB-` block includes evidence, minimum command or evidence, artifact, and pass condition |
| T-002 | R-010 | Documentation quality | `pattern='TB''D|et''c\.|as nee''ded|la''ter|prope''rly|so''me|f''ew|so''on'; rg -n "$pattern" docs/stable-major-release-readiness.md` | No matches are returned |
| T-003 | R-011 | Behavior safety | `git diff --name-only` | Changed files are documentation files unless a failing test justified code changes |
| T-004 | R-012 | Documentation discoverability | `rg -n "stable-major-release-readiness" docs/README.md` | The new readiness document is linked from the docs index |
| T-005 | R-007, R-011 | Unit verification | `go test ./...` | Command exits with code 0 |
| T-006 | R-007, R-011 | Static verification | `go vet ./...` | Command exits with code 0 |
| T-007 | R-005, R-008 | External consumer compile verification | Commands in section 13.4 | A fresh external module imports `simplykb`, references the stable API baseline, and compiles |

### 9. Failure Semantics and Recovery

| Scenario | Detection signal | Behavior | User-visible message | Recovery or rollback |
|---|---|---|---|---|
| F-001 | Required source file is missing | Hard fail | `Required source file is missing: <path>` | Stop and ask the user to restore the file or confirm a new source path |
| F-002 | `go test ./...` fails | Hard fail for release readiness | `Unit tests failed; stable release readiness is not verified` | Report failing package and do not claim release readiness |
| F-003 | `go vet ./...` fails | Hard fail for release readiness | `go vet failed; static verification is not clean` | Report vet output and do not claim release readiness |
| F-004 | Docker or ParadeDB is unavailable for `make verify` | Degrade for documentation-only task; hard fail for release tagging | `Full release verification was not completed because Docker or ParadeDB was unavailable` | Run `go test ./...` and `go vet ./...`; tell user to run `make verify` before any tag |
| F-005 | External module check fails | Hard fail for `v1.0.0` tagging | `External module verification failed; do not publish v1.0.0` | Fix module packaging or dependency issue, then repeat the external module check |
| F-006 | A public API flaw is found | Hard fail for `v1.0.0` tagging | `Public API stability blocker found: <summary>` | Add a regression test, fix the flaw, update docs, rerun verification |

### 10. Observability

#### 10.1 Logs

- L-001: Release readiness command output must include `go test ./...` result.
- L-002: Release readiness command output must include `go vet ./...` result.
- L-003: Full release readiness command output must include `make verify` result before final tagging.
- L-004: Schema or retrieval changes must include `make integration-benchmark` output or a written reason why benchmark comparison is not required.

#### 10.2 Metrics and Status Signals

- M-001: `go test ./...` exit code is 0.
- M-002: `go vet ./...` exit code is 0.
- M-003: `make verify` exit code is 0 before final stable major release tagging.
- M-004: External module check exits with code 0 before final stable major release tagging.
- M-005: Integration benchmark results are recorded when schema, retrieval, or setup behavior changes.

#### 10.3 Operator Diagnostics

- OP-001: `make doctor` must pass against the supported local ParadeDB setup before a stable major release tag.
- OP-002: `make smoke` must print indexed document lines and non-empty search hits before a stable major release tag.
- OP-003: Integration CI must pass on the target commit before a stable major release tag.

### 11. Rollout and Rollback

#### 11.1 Rollout sequence

1. Add stable major release readiness documentation.
2. Update the documentation index.
3. Run `go test ./...`.
4. Run `go vet ./...`.
5. Ask the user whether to run full release verification with Docker using `make verify`.
6. If preparing an actual stable candidate, update `CHANGELOG.md` and follow `/Users/liu_y/code/goProject/simplykb/RELEASING.md`.
7. Publish a stable candidate release before final `v1.0.0`.
8. Collect external consumer validation evidence.
9. Publish `v1.0.0` only if all release blockers pass.

#### 11.2 Rollback trigger and steps

- Trigger: The readiness document creates confusion, contradicts README scope, or fails verification.
- Steps:
  - Revert only `/Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md` if the new document is wrong.
  - Revert only the link added to `/Users/liu_y/code/goProject/simplykb/docs/README.md` if the new document is removed.
  - Keep unrelated files unchanged.
  - Rerun `go test ./...` and `go vet ./...` after rollback.

### 12. Traceability Matrix

| Requirement ID | Implementation step IDs | Test IDs | Observability IDs |
|---|---|---|---|
| R-001 | I-002 | T-001 | L-001 |
| R-002 | I-002 | T-001 | L-001 |
| R-003 | I-002 | T-001 | L-001, M-003 |
| R-004 | I-002 | T-001 | L-001 |
| R-005 | I-001, I-002 | T-001 | L-001 |
| R-006 | I-001, I-002 | T-001 | L-003, M-003 |
| R-007 | I-002, I-005 | T-001, T-005, T-006 | L-001, L-002, M-001, M-002 |
| R-008 | I-002 | T-001 | M-004 |
| R-009 | I-001, I-002 | T-001 | L-001 |
| R-010 | I-004 | T-002 | L-001 |
| R-011 | I-005 | T-003, T-005, T-006 | M-001, M-002 |
| R-012 | I-003 | T-004 | L-001 |

### 13. Stable Major Release Bar

#### 13.0 Public API and behavior baseline for `v1.0.0`

`v1.0.0` freezes the following public surface unless a documented breaking-change release is made after `v1.0.0`.

| Category | Frozen public surface | Compatibility promise | Source anchor |
|---|---|---|---|
| Construction | `New(ctx context.Context, cfg Config) (*Store, error)` | Existing valid configs keep constructing a store or return a documented error. | `/Users/liu_y/code/goProject/simplykb/store.go` |
| Store lifecycle | `(*Store).Close()`, `(*Store).Ping(ctx)`, `(*Store).Migrate(ctx)`, `(*Store).EnsureSchema(ctx)` | Lifecycle and schema readiness behavior remain callable without requiring a new service boundary. | `/Users/liu_y/code/goProject/simplykb/store.go` |
| Document writes | `(*Store).UpsertDocument(ctx, req)`, `(*Store).ReindexDocument(ctx, req)`, `(*Store).DeleteDocument(ctx, collection, documentID)` | Document write, rebuild, and delete semantics remain synchronous and deterministic. | `/Users/liu_y/code/goProject/simplykb/store.go` |
| Search | `(*Store).Search(ctx, req)`, `(*Store).SearchDetailed(ctx, req)` | Search remains available through keyword, vector, and hybrid modes, with diagnostics available through the detailed API. | `/Users/liu_y/code/goProject/simplykb/store.go` and `/Users/liu_y/code/goProject/simplykb/types.go` |
| Configuration | `Config` | Existing documented fields keep the same meaning; new fields must be optional and have safe defaults. | `/Users/liu_y/code/goProject/simplykb/config.go` |
| Extension points | `Embedder`, `QueryEmbeddingCacheKeyer`, `Splitter` | Existing user implementations continue compiling unless a future major version documents a break. | `/Users/liu_y/code/goProject/simplykb/types.go` |
| Request and response types | `UpsertDocumentRequest`, `DocumentStats`, `SearchRequest`, `SearchHit`, `SearchResponse`, `SearchDiagnostics`, `ChunkDraft` | Existing fields keep their meaning; added fields must not require users to change existing code. | `/Users/liu_y/code/goProject/simplykb/types.go` |
| Built-in helpers | `NewDefaultSplitter`, `DefaultSplitter`, `NewHashEmbedder`, `HashEmbedder` | Helpers remain suitable for examples, tests, and local evaluation. | `/Users/liu_y/code/goProject/simplykb/splitter.go` and `/Users/liu_y/code/goProject/simplykb/embedder_hash.go` |
| Search constants | `SearchModeHybrid`, `SearchModeKeyword`, `SearchModeVector`, `QueryEmbeddingCacheStatus*` | Existing constant values remain stable for callers that persist or compare them. | `/Users/liu_y/code/goProject/simplykb/types.go` |
| Behavior | Schema migration, document upsert, metadata-only refresh, no-op unchanged upsert, stable chunk ids, metadata filtering, query embedding cache opt-in | Existing documented behavior does not silently change across patch or minor releases after `v1.0.0`. | `/Users/liu_y/code/goProject/simplykb/README.md` |

#### 13.1 Release blockers for `v1.0.0`

- RB-001: Public API freeze is written down and reviewed.
  - Evidence: Documentation lists the exported API and behavior promise.
  - Minimum command: `rg -n "Public API|compatibility|v1.0.0" docs/stable-major-release-readiness.md RELEASING.md README.md`
  - Artifact: `/Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md`, section 13.0.
  - Pass condition: A reviewer can identify every frozen exported API and behavior promise without opening source files.
- RB-002: Migration reliability is verified for all released schema states.
  - Evidence: Integration tests cover legacy schema upgrade paths from released versions.
  - Minimum command: `SIMPLYKB_DATABASE_URL="<local-db-url>" go test ./... -run Integration -count=1 -v`
  - Artifact: Integration test output saved in release preparation notes.
  - Pass condition: Integration tests pass against the supported ParadeDB baseline.
- RB-003: Local developer path is verified.
  - Evidence: `make smoke` and `make doctor` pass.
  - Minimum command: `make verify`
  - Artifact: `make verify` output saved in release preparation notes.
  - Pass condition: `make verify` exits with code 0.
- RB-004: External module usage is verified.
  - Evidence: A fresh module can `go get github.com/LyleLiu666/simplykb@<candidate-tag>`, import the package, reference the stable API baseline, and build.
  - Minimum command: Run the commands in section 13.4.
  - Artifact: External module command output saved in release preparation notes.
  - Pass condition: The external module compiles and exits with code 0.
- RB-005: Real consumer feedback is collected for the stable candidate.
  - Evidence: At least one real or representative Go service imports the SDK and exercises migration, upsert, delete, reindex, and search.
  - Minimum evidence: A written note in release preparation materials with repository path or reproducible commands.
  - Artifact: `/Users/liu_y/code/goProject/simplykb/docs/release-validation/v1.0.0.md`.
  - Pass condition: The note names the consumer, the candidate tag, the commands run, the result, and every blocker found.
- RB-006: Benchmark comparison is recorded when schema, retrieval, or setup behavior changes.
  - Evidence: `make integration-benchmark` output is attached to the release preparation notes.
  - Minimum command: `make integration-benchmark`
  - Artifact: Benchmark output saved in release preparation notes.
  - Pass condition: Any meaningful regression is documented with an accept, fix, or defer decision.
- RB-007: Product boundaries are preserved.
  - Evidence: No new core SDK feature introduces PDF parsing, OCR, ACL policy, dashboard behavior, hosted service behavior, or async platform behavior.
  - Minimum command: `git diff --name-only` plus reviewer inspection.
  - Artifact: Release PR description or release preparation notes.
  - Pass condition: Reviewer confirms the diff keeps the SDK inside the README boundary.
- RB-008: Changelog and release notes answer setup, migration, search behavior, examples, docs, and breaking-change impact.
  - Evidence: `CHANGELOG.md` contains the release entry before tagging.
  - Minimum command: `rg -n "setup|migration|search|example|breaking" CHANGELOG.md`
  - Artifact: `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md`.
  - Pass condition: Release notes explicitly say whether setup, migration, search, examples, docs, or breaking changes are affected.

#### 13.2 Optional improvements that must not block `v1.0.0`

- OI-001: More ingestion examples can improve adoption, but the core SDK must not absorb a document ingestion platform.
- OI-002: More benchmark scenarios can improve confidence, but absence of broad benchmarks must not block release if the existing main paths are measured.
- OI-003: Reranking examples can help advanced users, but reranking should stay outside the core SDK until real workload evidence requires a core change.
- OI-004: More provider examples can help onboarding, but the SDK should not become provider-specific.

#### 13.3 Recommended release path

1. Keep the current `v0.2.1` state as the baseline.
2. Use `v0.3.0` if the next release contains normal incremental hardening.
3. Use `v0.9.0` only if the release is explicitly positioned as the final compatibility candidate before `v1.0.0`.
4. Run the full release checklist in `/Users/liu_y/code/goProject/simplykb/RELEASING.md`.
5. Ask at least one external consumer to test the stable candidate.
6. Fix only release blockers found during candidate validation.
7. Publish `v1.0.0` after every release blocker in section 13.1 passes.

#### 13.4 External consumer compile check

Run these commands from a temporary directory outside `/Users/liu_y/code/goProject/simplykb`.

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

The compile check may fail at runtime if a future agent changes it to require a live database. Keep this check compile-focused and do not require a live database in this external module smoke test.

### 14. Readiness Checklist

- [x] No unresolved placeholders are present.
- [x] Every requirement has implementation steps and verification items.
- [x] Every command is runnable from `/Users/liu_y/code/goProject/simplykb` unless the command explicitly creates a fresh external module.
- [x] Failure behavior is explicit for critical paths.
- [x] Rollback is explicit and limited to touched documentation files.

## Kickoff Instruction

Use this prompt to start a fresh implementation conversation with a lower-capability coding agent.

```text
You are implementing stable major release readiness documentation in repository /Users/liu_y/code/goProject/simplykb.

Primary objective:
- Define the exact bar for publishing a stable major release such as v1.0.0, and make the bar executable by a future maintainer.

Authoritative inputs, in priority order:
1. /Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md
2. /Users/liu_y/code/goProject/simplykb/RELEASING.md
3. /Users/liu_y/code/goProject/simplykb/README.md
4. /Users/liu_y/code/goProject/simplykb/CHANGELOG.md
5. /Users/liu_y/code/goProject/simplykb/Makefile
6. /Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml
7. /Users/liu_y/code/goProject/simplykb/.github/workflows/integration.yml

Done criteria:
- D-001: A stable major release checklist exists in repository documentation and can be followed without hidden context.
- D-002: The checklist separates must-have release blockers from optional improvements.
- D-003: Every must-have blocker maps to a concrete verification command or manual evidence item.
- D-004: The repository states the public API and behavior compatibility promise for a stable major release.
- D-005: The repository documents the recommended release path from the current v0.2.1 state to a stable major release candidate.
- D-006: go test ./... and go vet ./... pass after documentation or code changes.

Non-negotiable constraints:
- Follow TDD thinking: identify verification before changing behavior.
- Do not change unrelated files.
- Do not use destructive git operations.
- Do not publish tags or create releases.
- Do not add platform features such as PDF parsing, OCR, dashboards, ACL policy, hosted service behavior, or async worker systems to the core SDK.
- Do not use keyword matching or brittle trigger words as a substitute for semantic behavior in new runtime logic.
- Keep documentation plain and understandable for a non-expert reader.

Execution order:
1. Read /Users/liu_y/code/goProject/simplykb/docs/stable-major-release-readiness.md.
2. Read /Users/liu_y/code/goProject/simplykb/RELEASING.md.
3. Read /Users/liu_y/code/goProject/simplykb/README.md.
4. Check whether /Users/liu_y/code/goProject/simplykb/docs/README.md links to the readiness document.
5. If the readiness document is missing or incomplete, create or update it according to the Execution Spec inside the readiness document.
6. If the docs index does not link to the readiness document, add the link.
7. Run `pattern='TB''D|et''c\.|as nee''ded|la''ter|prope''rly|so''me|f''ew|so''on'; rg -n "$pattern" docs/stable-major-release-readiness.md` and remove vague wording if the command reports matches.
8. Run `rg -n "^- RB-[0-9]{3}:|  - Evidence:|  - Minimum command|  - Minimum evidence:|  - Artifact:|  - Pass condition:" docs/stable-major-release-readiness.md` and confirm every `RB-` block has evidence, minimum command or evidence, artifact, and pass condition.
9. If preparing a real candidate tag, run the external consumer compile check in section 13.4.
10. Run go test ./....
11. Run go vet ./....
12. Report changed files, verification results, and requirement coverage.

Command policy:
- Run commands from /Users/liu_y/code/goProject/simplykb unless a command explicitly creates a fresh external module.
- Prefer rg for search.
- Use deterministic, non-interactive commands.
- If a command fails, report the command, exit code, key error output, and next action.

Blocker protocol:
- Stop only if a required source file is missing.
- Stop only if requirements conflict and the priority order cannot resolve the conflict.
- Stop only if an external dependency is unavailable and no fallback verification exists.
- When blocked, report blocker type, exact evidence, attempted workaround, and the minimum question needed to continue.

Required final output format:
1. Implemented Changes
2. Verification Results
3. Requirement Coverage
4. Known Risks / Follow-ups

Requirement Coverage must use explicit mapping:
- R-001 -> I-002, T-001 (pass or fail)
- R-002 -> I-002, T-001 (pass or fail)
- R-003 -> I-002, T-001 (pass or fail)
- R-004 -> I-002, T-001 (pass or fail)
- R-005 -> I-001 and I-002, T-001 (pass or fail)
- R-006 -> I-001 and I-002, T-001 (pass or fail)
- R-007 -> I-002 and I-005, T-001, T-005, T-006 (pass or fail)
- R-008 -> I-002 and I-008, T-001 and T-007 (pass or fail)
- R-009 -> I-001 and I-002, T-001 (pass or fail)
- R-010 -> I-004, T-002 (pass or fail)
- R-011 -> I-005, T-003, T-005, T-006 (pass or fail)
- R-012 -> I-003, T-004 (pass or fail)

First response format:
1. Understanding: 3 to 6 bullets.
2. Assumptions: explicit bullets.
3. First Action: the exact first command to run.
```

## Readiness Gate Report

| Gate | Status | Evidence |
|---|---|---|
| Completeness Gate | PASS | Execution Spec, Kickoff Instruction, and Readiness Gate Report are present. |
| Traceability Gate | PASS | Requirements R-001 through R-012 map to implementation steps, tests, and observability in section 12. |
| Executability Gate | PASS | Commands are listed with repository-root execution context and expected evidence. |
| Ambiguity Gate | PASS | Requirements use stable IDs, concrete file paths, and explicit pass evidence. |
| Safety Gate | PASS | Rollback is limited to touched documentation files, and destructive Git operations are forbidden. |
