# simplykb External Feedback Comparison And Revision Plan

## Document Purpose

This document compares one external feedback note against the current `simplykb` repository.

The goal is not to argue with the feedback for the sake of argument.
The goal is to separate:

1. what is supported by the current repository
2. what may belong to an unknown consumer project instead of the SDK itself
3. what should be rewritten before the feedback is reused in docs, review comments, or external communication

## Scope And Assumptions

This plan evaluates the current `simplykb` repository only.

It does not assume that referenced files such as `backend/service.go` or `service_test.go` exist in this repository, because they do not.
Those file references likely belong to a separate business application that embeds `simplykb`.

That distinction matters.
If we mix "consumer project behavior" with "SDK behavior", the final judgment will sound more confident than the evidence actually allows.

## Validation Basis

The comparison below is based on:

- current repository source under [`store.go`](../../../store.go), [`schema.go`](../../../schema.go), [`config.go`](../../../config.go), [`embedder_hash.go`](../../../embedder_hash.go), and [`README.md`](../../../README.md)
- repository tests under [`integration_test.go`](../../../integration_test.go), [`store_test.go`](../../../store_test.go), and [`sdk_consumer_integration_test.go`](../../../sdk_consumer_integration_test.go)
- local verification run on 2026-04-21 with `go test ./...`
- local verification run on 2026-04-21 with `make verify`

The local verification passed, including:

- unit tests
- `go vet`
- quickstart smoke run
- integration tests against local ParadeDB
- external consumer integration simulation

Re-run verification before publication or reuse of this evaluation after any release change, CI change, verification workflow change, or material update to the referenced repository files.

## Item-By-Item Comparison

### 1. "接入方式 8/10"

Original point:

- the project is not tightly coupled to business logic
- the main path depends on a minimal surface such as `Search` and `UpsertDocument`
- startup migration, page indexing, and metadata filter pushdown are all wired through

Verified facts:

- the public integration shape in the current SDK is explicitly documented as `Migrate`, `UpsertDocument`, and `Search` in [`README.md`](../../../README.md)
- the actual SDK implementation exposes those behaviors in [`store.go`](../../../store.go)
- the repository has an external consumer integration test that creates a fresh Go module, imports `simplykb`, runs `Migrate`, `UpsertDocument`, and `Search`, and expects a successful result in [`sdk_consumer_integration_test.go`](../../../sdk_consumer_integration_test.go)

What is wrong or incomplete in the original point:

- the cited files do not belong to this repository
- the original wording mixes "SDK integration surface" with "one specific consumer project's service flow"
- the current SDK story is slightly stronger than the original wording suggests, because `Migrate` is also part of the intended integration contract, not just `Search` and `UpsertDocument`

Revision action:

- keep the high-level positive judgment
- rewrite the evidence so it cites the SDK repository itself
- explicitly separate "SDK API shape is small" from "a business project integrated it cleanly"

Recommended rewrite:

`simplykb` has a clean integration shape.
The repository consistently presents the core path as `Migrate`, `UpsertDocument`, and `Search`, and it includes an external consumer integration test to prove that a fresh Go module can embed the SDK successfully.
That supports a positive judgment on SDK integration simplicity, but it should not be presented as proof about any specific business project's internal service design unless that project is reviewed separately.

### 2. "SDK 成熟度 6/10，因为锁定 v0.1.0"

Original point:

- the SDK is pinned at `v0.1.0`
- that usually means boundaries are still moving
- future upgrades may still carry behavior change risk

Verified facts:

- the current repository has a `v0.1.0` tag
- the first release is recorded in [`CHANGELOG.md`](../../../CHANGELOG.md)
- the repository also has several positive maturity signals:
- CI with unit and integration coverage in [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml)
- a release guide in [`RELEASING.md`](../../../RELEASING.md)
- an external consumer compatibility test in [`sdk_consumer_integration_test.go`](../../../sdk_consumer_integration_test.go)
- a `make verify` path recorded in [`Makefile`](../../../Makefile)

What is wrong or incomplete in the original point:

- version number alone is too weak as the primary maturity argument
- "early version" is true, but "therefore only 6/10" is more of a judgment call than a verified conclusion
- the original point misses the repository's visible engineering discipline around verification and release hygiene

Revision action:

- keep the "still early" caution
- soften the maturity conclusion so it reflects both risk and positive signals
- replace "locked to v0.1.0, therefore risky" with "still early, but already has credible release and verification scaffolding"

Recommended rewrite:

`simplykb` is still an early SDK and should not be described as battle-hardened yet.
The current public release is `v0.1.0`, so upgrade and behavior stability should still be watched carefully.
That said, the repository already shows healthier maturity signals than a raw version number suggests: CI covers integration, there is a release guide, there is a `make verify` flow, and an external consumer test checks that the SDK can be embedded from a fresh module.

### 3. "工程质量不错，索引失败会回滚页面写入；还有 backfill 和降级"

Original point:

- indexing failure rolls back page writes to avoid dirty state
- historical pages have backfill
- backfill failure degrades gracefully instead of taking the whole site down

Verified facts:

- `UpsertDocument` in [`store.go`](../../../store.go) wraps document upsert, old chunk deletion, new chunk insertion, and chunk count update in one transaction
- that does support the claim that SDK-level write consistency is handled deliberately
- integration tests cover several failure paths such as splitter failure, vector count mismatch, vector dimension mismatch, and embedding dimension drift in [`integration_test.go`](../../../integration_test.go)

What is wrong or unsupported in the original point:

- "page write rollback" is directionally similar to SDK transactional consistency, but the wording sounds like a page-oriented business application rather than the SDK itself
- "historical page backfill" is not a verified feature in the current SDK repository
- "graceful degradation that does not take the whole site down" is also not established by the current SDK repository alone

Revision action:

- keep the transactional consistency praise
- remove or quarantine backfill and whole-site degradation claims unless the consumer project is reviewed directly
- rewrite this item so it praises what the SDK actually proves today

Recommended rewrite:

The SDK shows solid engineering judgment around write consistency.
`UpsertDocument` is transactional, so document metadata and chunk index updates do not silently drift apart on partial failure.
The repository also includes failure-path integration coverage for splitter output, vector shape mismatch, and schema dimension drift.
Claims about page backfill or application-wide graceful degradation should only remain if they are verified in the consumer project that embeds the SDK.

### 4. "业务方还会二次过滤，说明还不敢完全信 SDK"

Original point:

- after receiving search results, the business side filters again against page data
- this supposedly means the SDK's own filtering is not fully trusted yet

Verified facts:

- the SDK pushes metadata filtering down into both keyword and vector SQL paths in [`store.go`](../../../store.go)
- the repository includes integration coverage for metadata filtering in [`integration_test.go`](../../../integration_test.go)

What is wrong or too strong in the original point:

- a second business-side validation step does not automatically prove distrust in the SDK
- it may just reflect defense-in-depth, permission checks, stale object protection, or other domain-specific safety rules
- the SDK repository itself already provides direct evidence that metadata filtering exists and is tested

Revision action:

- remove the implied accusation
- rewrite this as an inference with lower confidence
- distinguish "SDK supports metadata filter pushdown" from "consumer project may still keep a defensive recheck"

Recommended rewrite:

The SDK already supports metadata filter pushdown and has integration coverage for it.
If a consumer project performs an additional result check after search, that should be described as defense-in-depth unless there is direct evidence that the extra step exists because the SDK filter is known to be unreliable.

### 5. "本地默认 hash embedder 很务实，但生产环境直接禁止继续用 hash"

Original point:

- the local default uses `HashEmbedder`
- that is good for getting started quickly
- semantic quality is limited
- production use is explicitly blocked

Verified facts:

- the quickstart and README use `HashEmbedder` for local evaluation in [`README.md`](../../../README.md)
- the README explicitly labels it demo-only and says not to treat it as a production strategy in [`README.md`](../../../README.md)
- a more production-shaped provider example exists in [`examples/openai_compatible/main.go`](../../../examples/openai_compatible/main.go)
- the SDK does not appear to contain a runtime environment check that forbids `HashEmbedder` in production

What is wrong or incomplete in the original point:

- "production use is directly forbidden" overstates what the code enforces
- the repository draws a strong documentation boundary between demo and production
- that is helpful and honest, but it is not the same thing as a hard runtime guard

Revision action:

- keep the praise for the demo path
- keep the warning that semantic quality is limited
- change "hard block in production" to "strongly discouraged by docs and examples"

Recommended rewrite:

Using `HashEmbedder` in local demos is a practical choice because it keeps the first-run path cheap and fast.
The repository is honest that this is only a smoke and demo tool, and it provides a more production-shaped provider example for real deployments.
However, this boundary is currently enforced by documentation and example design rather than by a hard runtime production guard.

## Cross-Cutting Revision Rules

The full rewrite should follow these rules:

1. Split SDK facts from consumer-project facts.
2. Treat missing repository files as a scope warning, not as evidence.
3. Prefer repository-native citations when evaluating the SDK itself.
4. Use softer language for inference and stronger language for verified behavior.
5. Do not turn defensive business logic into automatic evidence that the SDK is weak.
6. Do not describe documentation guidance as if it were runtime enforcement.

## Execution Spec

### 0. Metadata

- Task name: Upgrade external feedback comparison plan into a low-ambiguity execution handoff package
- Date: 2026-04-21
- Owner: Codex
- Source inputs:
  - `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
  - `/Users/liu_y/.codex/skills/execution-handoff/references/spec-template.md`
  - `/Users/liu_y/.codex/skills/execution-handoff/references/kickoff-template.md`
  - `/Users/liu_y/code/goProject/simplykb/README.md`
  - `/Users/liu_y/code/goProject/simplykb/store.go`
  - `/Users/liu_y/code/goProject/simplykb/integration_test.go`
  - `/Users/liu_y/code/goProject/simplykb/sdk_consumer_integration_test.go`
  - `/Users/liu_y/code/goProject/simplykb/Makefile`
  - `/Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml`
  - `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md`

### 1. Mission and Done Definition

#### 1.1 Mission

Convert this document from a human-readable comparison memo into a self-contained execution package that a lower-capability coding agent can use to update or recreate the external evaluation with minimal interpretation risk.

#### 1.2 Done Criteria

- D-001: The document contains explicit requirements, ordered implementation steps, and verification steps.
- D-002: Every reusable external summary begins with a scope guard and ends with an evidence note.
- D-003: The document contains a freshness rule for time-sensitive repository facts.
- D-004: The document contains a quality gate that requires citations for facts, labels for inferences, and code-backed wording for runtime enforcement claims.
- D-005: The document contains a kickoff instruction that a new conversation can follow without hidden context.
- D-006: The document contains a readiness gate report with explicit PASS or FAIL results and concrete fixes applied.

### 2. Scope

#### 2.1 In Scope

- S-001: Update `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`.
- S-002: Preserve the existing item-by-item comparison as rationale for future edits.
- S-003: Add a formal execution package for a lower-capability coding agent.
- S-004: Preserve the reusable revised summary with the correct scope and evidence guardrails.

#### 2.2 Out of Scope

- OOS-001: Modify `/Users/liu_y/code/goProject/simplykb/README.md`.
- OOS-002: Modify Go source files under `/Users/liu_y/code/goProject/simplykb`.
- OOS-003: Review or edit a separate consumer repository that embeds `simplykb`.
- OOS-004: Re-score the SDK beyond what is already supported by repository-native evidence.

These out-of-scope items apply to this handoff-package task itself.
If the user explicitly asks to move from handoff completion into repository implementation, adjacent repository files such as README, workflow files, and GitHub templates may be updated under the relevant design document instead of treating this memo as the only implementation source.

#### 2.3 Non-goals

- NG-001: Do not turn this document into a broad product roadmap.
- NG-002: Do not invent claims about `backend/service.go`, `service_test.go`, or any other missing consumer-project files.
- NG-003: Do not describe documentation guidance as if it were runtime enforcement.

### 3. Constraints and Assumptions

#### 3.1 Hard Constraints

- C-001: Edit only `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`.
- C-002: Use repository-native citations or dated local verification commands for retained factual claims.
- C-003: Preserve the distinction between the `simplykb` SDK repository and any separate consumer project.
- C-004: Preserve the freshness rule for time-sensitive repository facts.
- C-005: Do not use destructive git operations.

#### 3.2 Assumptions

- A-001: The user wants this same document to become the handoff artifact.
- A-002: The repository files listed in the validation basis remain the authoritative evidence set unless newer verification replaces them.
- A-003: No code behavior changes are required for this task because the target is a documentation handoff package.
- A-004: If the user explicitly requests follow-on repository implementation after the handoff package is complete, the execution source should broaden to the repository design docs that define concrete repo changes.

### 4. Canonical Terms

| Term | Definition | Allowed aliases | Disallowed aliases |
|---|---|---|---|
| `SDK repository` | The current `simplykb` repository under `/Users/liu_y/code/goProject/simplykb` | repo, current repository | backend project, consumer service |
| `consumer project` | A separate application that embeds `simplykb` | embedding app, downstream app | the SDK itself |
| `scope guard` | An explicit first sentence that states what system the evaluation covers | scope sentence | implied scope |
| `evidence note` | A short note listing the repository files and verification commands behind a reusable summary | evidence summary | vague “based on review” wording |
| `runtime enforcement` | Behavior prevented or required by code or tests | hard guard, enforced in code | docs recommendation |
| `documentation guidance` | Advice or warnings that are not enforced by code | docs-only boundary, guidance | hard block |

### 5. Requirement List

| ID | Type | Requirement statement | Priority | Rationale |
|---|---|---|---|---|
| R-001 | functional | Add an execution-ready section that defines mission, done criteria, scope, constraints, canonical terms, and explicit requirements for this document update. | must | A lower-capability agent needs structured instructions, not only narrative advice. |
| R-002 | functional | Provide ordered implementation steps that point only to `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`. | must | The task is document-only and must not drift into unrelated files. |
| R-003 | functional | Provide explicit verification commands and pass criteria for the critical handoff requirements. | must | The agent must be able to prove completion without subjective judgment. |
| R-004 | functional | Preserve a reusable external summary that starts with a scope guard and ends with an evidence note. | must | Copy-paste reuse is a primary risk surface for this document. |
| R-005 | non-functional | Preserve a freshness rule requiring re-verification after release, CI, verification workflow, or cited-file changes. | must | The summary relies on time-sensitive repository facts. |
| R-006 | non-functional | Preserve a quality gate that requires citations for facts, labels for inferences, and code-backed wording for runtime enforcement claims. | must | The original failure mode was overclaiming beyond evidence. |
| R-007 | functional | Add a kickoff instruction for a fresh conversation with authoritative inputs, execution order, blocker protocol, and final output format. | must | The lower-capability agent must be able to start without hidden context. |
| R-008 | functional | Keep the existing item-by-item comparison sections as rationale for why each rewrite rule exists. | should | Removing the rationale would make future edits easier to drift. |
| R-009 | safety | Define hard-fail conditions and rollback behavior for missing inputs, stale facts, unrelated file edits, and uncited runtime-enforcement claims. | must | A lower-capability agent needs explicit stop conditions. |
| R-010 | non-functional | Include a readiness gate report with PASS or FAIL status per gate and the exact fixes applied. | must | The handoff package needs an explicit shipment check. |

### 6. Interface and Data Contracts

#### 6.1 Inputs

| Input | Source | Type | Validation | Failure behavior |
|---|---|---|---|---|
| target document | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Markdown file | `test -f` returns exit code 0 | hard-fail |
| execution spec template | `/Users/liu_y/.codex/skills/execution-handoff/references/spec-template.md` | Markdown file | `test -f` returns exit code 0 | hard-fail |
| kickoff template | `/Users/liu_y/.codex/skills/execution-handoff/references/kickoff-template.md` | Markdown file | `test -f` returns exit code 0 | hard-fail |
| repository evidence files | `README.md`, `store.go`, `integration_test.go`, `sdk_consumer_integration_test.go`, `Makefile`, `.github/workflows/ci.yml`, `CHANGELOG.md` | source files | each `test -f` returns exit code 0 | hard-fail |

#### 6.2 Outputs

| Output | Consumer | Type | Success criteria |
|---|---|---|---|
| updated handoff document | user or lower-capability coding agent | Markdown file | contains execution spec, kickoff instruction, readiness gate report, and reusable revised summary |
| reusable revised summary | reviewer, doc author, external communicator | prose section | starts with a scope guard and ends with an evidence note |
| readiness gate report | user or future agent | structured Markdown section | every gate shows PASS or FAIL with concrete fixes applied |

#### 6.3 State Transitions

| From | Event | Guard | To | Notes |
|---|---|---|---|---|
| comparison memo | execution package sections added | all required headings exist | execution-ready draft | this is the first structural upgrade |
| execution-ready draft | verification commands pass | scope guard, evidence note, freshness rule, and traceability all present | handoff-ready document | this is the ship-ready state |

### 7. Implementation Plan

| Step ID | Goal | Files | Change action | Command(s) | Expected evidence |
|---|---|---|---|---|---|
| I-001 | Confirm all authoritative inputs exist before editing | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`, `/Users/liu_y/.codex/skills/execution-handoff/references/spec-template.md`, `/Users/liu_y/.codex/skills/execution-handoff/references/kickoff-template.md`, `/Users/liu_y/code/goProject/simplykb/README.md`, `/Users/liu_y/code/goProject/simplykb/store.go`, `/Users/liu_y/code/goProject/simplykb/integration_test.go`, `/Users/liu_y/code/goProject/simplykb/sdk_consumer_integration_test.go`, `/Users/liu_y/code/goProject/simplykb/Makefile`, `/Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml`, `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md` | Verify file presence and stop on any missing input | `test -f /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md && test -f /Users/liu_y/.codex/skills/execution-handoff/references/spec-template.md && test -f /Users/liu_y/.codex/skills/execution-handoff/references/kickoff-template.md && test -f /Users/liu_y/code/goProject/simplykb/README.md && test -f /Users/liu_y/code/goProject/simplykb/store.go && test -f /Users/liu_y/code/goProject/simplykb/integration_test.go && test -f /Users/liu_y/code/goProject/simplykb/sdk_consumer_integration_test.go && test -f /Users/liu_y/code/goProject/simplykb/Makefile && test -f /Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml && test -f /Users/liu_y/code/goProject/simplykb/CHANGELOG.md` | command exits 0 |
| I-002 | Add the formal execution package skeleton | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Replace the informal end section with Execution Spec subsections 0 through 6 | edit file with `apply_patch` | headings for metadata, mission, scope, constraints, canonical terms, requirements, and interfaces appear |
| I-003 | Add explicit execution and verification flow | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Insert implementation plan, verification plan, failure semantics, observability, rollout and rollback, traceability matrix, and readiness checklist | edit file with `apply_patch` | `I-*`, `T-*`, `F-*`, `L-*`, `M-*`, and `OP-*` identifiers appear |
| I-004 | Add a new-conversation kickoff instruction | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Insert a kickoff section with goal, inputs, execution order, blocker protocol, output format, and first-response format | edit file with `apply_patch` | `## Kickoff Instruction` appears with all required subsections |
| I-005 | Preserve the reusable summary with its guardrails | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Keep the revised summary, scope guard, freshness rule, and evidence note aligned with the new package | edit file with `apply_patch` | the summary still starts with the scope guard and ends with the evidence note |
| I-006 | Verify structure and safety before handoff | `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | Run deterministic checks for headings, IDs, and diff scope | `rg -n "^## Execution Spec|^## Kickoff Instruction|^## Readiness Gate Report|^## Ready-To-Use Revised Summary" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md && rg -n 'R-001|I-001|T-001|This evaluation covers the current \`simplykb\` repository itself|Evidence note:|Re-run verification before publication or reuse' /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md && git diff -- /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | required headings and guardrails exist; diff is limited to the target file |

### 8. Verification Plan

| Test ID | Covers requirement IDs | Type | Command | Pass criteria |
|---|---|---|---|---|
| T-001 | R-001, R-002 | structure | `rg -n "^## Execution Spec|^#### 1\\.1 Mission|^### 7\\. Implementation Plan" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | all three headings are found |
| T-002 | R-003, R-010 | structure | `rg -n "^\\| R-001|^\\| I-001|^\\| T-001|PASS|FAIL" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | requirement, implementation, verification, and gate markers are found |
| T-003 | R-004, R-006 | content | `rg -n 'This evaluation covers the current \`simplykb\` repository itself|Evidence note:|documentation guidance|runtime enforcement' /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | scope guard, evidence note, and enforcement distinction are found |
| T-004 | R-005 | content | `rg -n "Re-run verification before publication or reuse" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | the freshness rule is found |
| T-005 | R-007 | structure | `rg -n "^## Kickoff Instruction|^### 1\\. Role and Goal|^### 6\\. Blocker Protocol|^### 8\\. First Response Format" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | all kickoff subsections are found |
| T-006 | R-008, R-009 | safety | `rg -n "^## Item-By-Item Comparison|Do not change unrelated files|Do not use destructive git operations|Rollback trigger and steps" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` | rationale section and safety phrases are found |

### 9. Failure Semantics and Recovery

| Scenario | Detection signal | Behavior | User-visible message | Recovery/rollback |
|---|---|---|---|---|
| F-001 | any authoritative input file is missing | hard-fail | `Missing required input file: <absolute-path>` | stop; do not edit the target document until the missing file is resolved |
| F-002 | a cited repository fact no longer matches the current repo state | hard-fail for publication or reuse | `Repository facts changed; re-run verification and update validation basis before reuse` | update the validation basis, rerun `go test ./...` or `make verify` if needed, then refresh affected prose |
| F-003 | a required execution package heading or ID is missing after edits | hard-fail | `Execution package is incomplete: missing <section-or-id>` | continue editing the target document until the missing section is present, then rerun verification |
| F-004 | `git diff` shows unrelated file changes | hard-fail | `Unexpected file changes detected outside the target document` | revert only the unrelated edits manually with `apply_patch`; do not use destructive git commands |
| F-005 | a statement describes runtime enforcement but only docs support it | hard-fail | `Runtime enforcement claim is not backed by code or tests` | rewrite the statement as documentation guidance or add the missing code-backed citation |

### 10. Observability

#### 10.1 Logs

- L-001: Report the absolute path of every changed file in the final response.
- L-002: Report every verification command run and whether it passed.

#### 10.2 Metrics and Status Signals

- M-001: The number of `R-*` requirements matches the number of traceability rows.
- M-002: The readiness gate report shows PASS for all gates.

#### 10.3 Operator Diagnostics

- OP-001: Run `rg -n "^## " /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` to inspect top-level structure.
- OP-002: Run `git diff -- /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` to inspect only target-document changes.

### 11. Rollout and Rollback

#### 11.1 Rollout sequence

1. Verify authoritative input files exist.
2. Edit only `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`.
3. Run the verification commands in Section 8.
4. Inspect the target-file diff.
5. Hand the updated document to the user or lower-capability agent.

#### 11.2 Rollback trigger and steps

- Trigger: Any verification command fails, any unrelated file changes appear, or the reusable summary loses the scope guard or evidence note.
- Steps:
  - Manually revert only the incorrect hunks in `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` with `apply_patch`.
  - Re-run the verification commands in Section 8.
  - Do not use `git reset --hard`, `git checkout --`, or other destructive git operations.

### 12. Traceability Matrix

| Requirement ID | Implementation step IDs | Test IDs | Observability IDs |
|---|---|---|---|
| R-001 | I-002 | T-001 | L-001/M-001 |
| R-002 | I-002, I-003 | T-001 | L-001/OP-002 |
| R-003 | I-003, I-006 | T-002 | L-002/M-002 |
| R-004 | I-005, I-006 | T-003 | L-002/OP-001 |
| R-005 | I-005 | T-004 | L-002/OP-001 |
| R-006 | I-002, I-005 | T-003 | L-002/M-002 |
| R-007 | I-004 | T-005 | L-001/OP-001 |
| R-008 | I-005 | T-006 | OP-001 |
| R-009 | I-003, I-006 | T-006 | L-002/OP-002 |
| R-010 | I-003, I-006 | T-002 | M-002 |

### 13. Readiness Checklist

- [x] No placeholders such as `TBD`, `etc`, `as needed`, or `later` remain in the execution package.
- [x] Every reusable external blurb begins with an explicit scope guard stating whether it covers the `simplykb` repository itself or a separate consumer project.
- [x] Every retained factual claim is backed by a repository-native citation or a dated local verification command.
- [x] Every inference is presented as an inference instead of a settled fact.
- [x] Every statement about runtime enforcement is backed by code or tests; documentation-only boundaries are described as guidance, not enforcement.
- [x] Every externally reused version includes a short evidence note listing the repository files and verification commands it relies on.
- [x] Verification is re-run before publication or reuse whenever release state, CI behavior, verification workflow, or the cited repository files materially change.

## Kickoff Instruction

### 1. Role and Goal

You are updating `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md` inside repository `/Users/liu_y/code/goProject/simplykb`.

Primary objective:

- Convert the target document into a low-ambiguity handoff package that a lower-capability coding agent can use without hidden context.

Done criteria:

- D-001: The target document contains explicit requirements, ordered steps, and verification commands.
- D-002: The reusable revised summary starts with a scope guard and ends with an evidence note.
- D-003: The target document contains a freshness rule and a quality gate for evidence and inference wording.
- D-004: The target document contains a kickoff instruction and a readiness gate report.

### 2. Authoritative Inputs

Use only these inputs as source of truth:

- `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
- `/Users/liu_y/code/goProject/simplykb/README.md`
- `/Users/liu_y/code/goProject/simplykb/store.go`
- `/Users/liu_y/code/goProject/simplykb/integration_test.go`
- `/Users/liu_y/code/goProject/simplykb/sdk_consumer_integration_test.go`
- `/Users/liu_y/code/goProject/simplykb/Makefile`
- `/Users/liu_y/code/goProject/simplykb/.github/workflows/ci.yml`
- `/Users/liu_y/code/goProject/simplykb/CHANGELOG.md`
- `/Users/liu_y/.codex/skills/execution-handoff/references/spec-template.md`
- `/Users/liu_y/.codex/skills/execution-handoff/references/kickoff-template.md`

If inputs conflict, use this priority order:

1. current repository facts and dated verification commands
2. `/Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
3. execution handoff templates

### 3. Non-negotiable Constraints

- Follow TDD in document form: identify missing sections first, then edit, then verify.
- Do not change unrelated files.
- Do not use destructive git operations unless explicitly requested.
- Keep the distinction between the `simplykb` SDK repository and any separate consumer project explicit.
- Do not invent missing repository facts.
- If a blocker is found, stop and report exact file and line evidence.

### 4. Execution Order

1. Confirm understanding in 3 to 6 bullets.
2. List assumptions and mark risky assumptions.
3. Identify the missing execution-package sections in the target document.
4. Execute the document update in this exact order:
   - add or update the Execution Spec
   - add or update the Kickoff Instruction
   - add or update the Readiness Gate Report
   - verify the reusable revised summary still has the scope guard and evidence note
5. Run these verification commands:
   - `rg -n "^## Execution Spec|^## Kickoff Instruction|^## Readiness Gate Report|^## Ready-To-Use Revised Summary" /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
   - `rg -n 'R-001|I-001|T-001|This evaluation covers the current \`simplykb\` repository itself|Evidence note:|Re-run verification before publication or reuse' /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
   - `git diff -- /Users/liu_y/code/goProject/simplykb/docs/plans/2026-04-21-external-feedback-comparison-plan.md`
6. Report the result with changed file paths, verification results, and requirement coverage.

### 5. Command Policy

- Prefer deterministic, non-interactive commands.
- Prefer `rg` for search.
- Summarize command output instead of dumping irrelevant lines.
- If a command fails, include the command, exit code, key stderr, and next action.

### 6. Blocker Protocol

Stop and ask for input only when:

- a required input file is missing
- repository facts conflict and the conflict cannot be resolved by the declared priority order
- verification rules require a fresh command that cannot be run in the current environment

When blocked, report:

- blocker type
- exact evidence
- attempted workaround
- the minimum question needed to continue

### 7. Required Final Output Format

Return sections in this order:

1. `Implemented Changes`
2. `Verification Results`
3. `Requirement Coverage`
4. `Known Risks / Follow-ups`

`Requirement Coverage` must use explicit mapping:

- `R-001 -> I-002, T-001 (pass or fail)`
- `R-002 -> I-002, T-001 (pass or fail)`

### 8. First Response Format

The first response in the new conversation must include:

1. `Understanding`
2. `Assumptions`
3. `First Action`

## Readiness Gate Report

- Completeness Gate: PASS
  - Fixes applied: added Execution Spec, Kickoff Instruction, Readiness Gate Report, and preserved the reusable revised summary.
- Traceability Gate: PASS
  - Fixes applied: added stable requirement IDs, implementation step IDs, verification IDs, and a traceability matrix with no empty cells.
- Executability Gate: PASS
  - Fixes applied: added ordered commands, absolute file paths, explicit done criteria, and verification commands for the critical requirements.
- Ambiguity Gate: PASS
  - Fixes applied: added canonical terms, authoritative input priority, blocker protocol, and code-versus-guidance wording rules.
- Safety Gate: PASS
  - Fixes applied: restricted edits to one document, added hard-fail conditions, and added a non-destructive rollback procedure.

## Ready-To-Use Revised Summary

If we want a concise replacement paragraph, this is the recommended direction:

This evaluation covers the current `simplykb` repository itself, not a separate consumer application that embeds it.

I would rate the SDK integration shape positively.
`simplykb` keeps the core path small and explicit around `Migrate`, `UpsertDocument`, and `Search`, and the repository includes an external consumer integration test to show that a fresh Go module can embed the SDK successfully.

I would describe maturity as "early but disciplined" rather than reducing it to the `v0.1.0` label alone.
The current release is still young, so stability should still be watched carefully, but the repository already has meaningful trust signals such as CI integration coverage, a release guide, a `make verify` path, and failure-path tests.

I would keep the praise for engineering quality, but I would narrow it to what the repository actually proves today.
The SDK handles write consistency transactionally, checks embedding dimension drift during migration, and covers multiple failure paths in integration tests.
Claims about page backfill or broader application-level graceful degradation should remain only if they are verified in the consumer project being reviewed.

I would also soften the interpretation of any business-side second filtering.
The SDK itself already pushes metadata filters into the search queries and tests that behavior.
An extra business-side check is better described as defense-in-depth unless there is direct evidence that it exists because the SDK filter cannot be trusted.

Finally, the current repository is honest about `HashEmbedder`.
It is useful for smoke testing and demos, but the production boundary is expressed through docs and example design, not through a hard runtime block.

Evidence note: this summary is based on [`README.md`](../../../README.md), [`store.go`](../../../store.go), [`integration_test.go`](../../../integration_test.go), [`sdk_consumer_integration_test.go`](../../../sdk_consumer_integration_test.go), [`Makefile`](../../../Makefile), [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml), [`CHANGELOG.md`](../../../CHANGELOG.md), and local verification run on 2026-04-21 with `go test ./...` and `make verify`.
