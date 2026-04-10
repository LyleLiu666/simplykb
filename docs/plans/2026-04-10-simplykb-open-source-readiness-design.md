# simplykb Open Source Readiness Plan

## Document Purpose

This document defines what "excellent and out-of-the-box" should mean for `simplykb` as a public open source project.

This is not a vague wish list.
This is a concrete execution document for turning the repository into something that most developers can:

1. discover
2. understand
3. run locally
4. trust
5. evaluate quickly
6. adopt with low fear

The document is intentionally detailed.
The goal is to remove ambiguity, reduce drift, and make future work easier to execute.

## Problem Statement

`simplykb` already has a strong core shape:

- narrow product scope
- simple architecture
- local development path
- deterministic data model
- clear public API
- real tests

That is a strong foundation.
However, a public repository is judged by more than code quality.

For most users, an excellent open source project feels good because it answers these questions quickly:

- What is this?
- Why should I use it?
- Can I run it in 10 minutes?
- What does success look like?
- What are the limits?
- Is it actively maintained?
- What happens when something goes wrong?
- Can I trust the examples?
- Can I use it in a real project without reading the whole source tree?

If the answer to any of these questions is slow, unclear, or hidden in code, the project stops feeling "out of the box", even if the implementation itself is strong.

## Intended Audience

This plan targets three audience groups.

### Audience A: Curious Evaluators

These users arrive from GitHub, a shared link, or a recommendation.
They want to know in a few minutes whether the project is worth trying.

They care about:

- crisp positioning
- quick start clarity
- screenshots or concrete outputs
- confidence that the repo is alive

### Audience B: Practical Builders

These users want to integrate `simplykb` into a Go service or internal tool.
They care about:

- setup reliability
- contract stability
- examples with realistic usage
- operational boundaries
- upgrade safety

### Audience C: Contributors

These users may report issues, improve docs, or add features.
They care about:

- repo conventions
- test expectations
- CI behavior
- release process
- architecture boundaries

An excellent repository helps all three groups without forcing any one group to reverse-engineer the author's intent.

## Definition Of Success

For `simplykb`, an excellent out-of-the-box open source project means:

1. A new user can run the quickstart in 10 minutes or less on a normal development machine.
2. The README explains the value, limitations, and first steps without requiring code reading.
3. The project has visible trust signals: CI, release tags, clear status, contribution guidance.
4. The repository makes production boundaries explicit instead of hinting at more than it supports.
5. The project fails early and clearly when callers violate important contracts.
6. Example code reflects realistic usage patterns rather than demo-only shortcuts.
7. A contributor can make a change and know how to verify it.

If these conditions are true, most users will describe the project as "clean", "serious", "easy to try", and "safe to evaluate".

## Current State Assessment

This section evaluates the current repository as it exists today.

### What Is Already Strong

- The scope is intentionally narrow and this is explained well.
- The core package API is small.
- The local story is understandable: Docker Compose plus Go SDK.
- The architecture has a believable production shape for single-node workloads.
- The code reads as deliberate rather than chaotic.
- The design doc already states non-goals, which prevents false expectations.
- Tests exist for the main logic and integration flow.

### What Is Still Missing For Public Excellence

- The README is good, but it still reads more like a strong internal note than a world-class public landing page.
- There is no visible CI status in the repository workflow yet.
- There is no release policy or versioned public promise.
- There is no contributor guide.
- There is no issue template or discussion structure.
- There is no realistic provider example beyond the hash embedder.
- There is limited "what success should look like" output in the docs.
- There is no compatibility matrix for Go, ParadeDB, and expected environment.
- There is not yet a formal checklist for public release hygiene.

### Overall Assessment

Current state:

- product clarity: strong
- first-run experience: good
- trust and maintainability signals: moderate
- contributor friendliness: moderate to weak
- adoption confidence for external teams: moderate

Summary judgment:

`simplykb` is already a promising public project, but not yet a polished "excellent out-of-the-box" public repository for most users.

## What Excellent Means Specifically For simplykb

The project should not try to win by being the biggest knowledge base system.
It should win by being the clearest, safest, and most believable small one.

That means the excellence standard is:

- short, but complete
- narrow, but trustworthy
- easy to start, but explicit about limits
- production-minded, but not pretending to be a platform

This distinction matters.
Many open source projects become weaker when they chase breadth too early.
For `simplykb`, excellence comes from disciplined sharpness, not feature sprawl.

The repository should feel like:

"This project knows exactly what it is, exactly what it is not, and it helps me succeed quickly if my use case fits."

That is the target feeling.

## Approach Options

There are three reasonable ways to improve the repository.

### Option A: Minimal Polish

Scope:

- improve README
- add CI
- add contributor guide
- tag first release

Advantages:

- fast
- low risk
- immediate improvement in public appearance

Disadvantages:

- still weak on real-world adoption confidence
- still relies too much on the reader inferring usage details

### Option B: Recommended Balanced Path

Scope:

- everything in Option A
- add realistic provider example
- add troubleshooting and compatibility notes
- add release and versioning policy
- add repository hygiene templates
- define acceptance checks for public changes

Advantages:

- strongest improvement per unit of work
- raises both usability and trust
- keeps the project small while making it feel mature

Disadvantages:

- requires focused documentation work
- needs some maintenance discipline after the initial pass

### Option C: Ambitious Public Package Push

Scope:

- everything in Option B
- benchmark docs
- richer example apps
- tutorial series
- comparison pages
- templates for multiple providers

Advantages:

- strongest public positioning
- better for broad awareness

Disadvantages:

- high maintenance burden
- higher risk of overpromising
- may pull the project away from its low-entropy philosophy

## Recommended Direction

Use Option B.

Option B is the right fit because it improves the first-run experience, public trust, and contributor clarity without turning `simplykb` into a documentation-heavy marketing project.

In simple terms:

- do not stay too light
- do not become bloated
- make the repo feel dependable

That is the sweet spot.

## Scope Lock

### In Scope

- repository landing experience
- quickstart clarity
- trust signals
- contributor onboarding
- public release hygiene
- example realism
- versioning and change communication
- quality gates for public changes

### Out Of Scope

- adding many product features
- turning the SDK into a hosted service
- multi-tenant platform concerns
- dashboard work
- workflow orchestration
- benchmark competition work

### Non-Goals

- becoming the most feature-rich RAG system
- supporting every embedding provider immediately
- documenting unrelated Postgres or Go basics
- optimizing for highly specialized enterprise requirements

## Execution Spec

This section defines explicit requirements.
Each requirement has:

- a stable ID
- a clear goal
- a concrete deliverable
- a verification method

### Theme 1: Landing Experience

#### R-001 Repository homepage must explain value in 30 seconds

Requirement:

- Rewrite the top of [README.md](../../README.md) so a first-time visitor can understand the project in under 30 seconds.
- The first screen must answer:
  - what `simplykb` is
  - who it is for
  - why it exists
  - when not to use it

Deliverables:

- stronger opening paragraph
- short "best fit" bullets
- short "not for you if" bullets

Verification:

- a reviewer can summarize project purpose after reading only the top section
- the README top section does not depend on code examples for meaning

#### R-002 README must show a clear success path

Requirement:

- Add a numbered first-run flow to [README.md](../../README.md).
- The flow must include:
  - start database
  - run example
  - expected output
  - how to run tests

Deliverables:

- explicit commands
- one illustrative sample output block labeled as example shape, not exact literal output
- one sentence per step explaining what success means
- one list of stable success signals

Verification:

- a new user can compare the local run against stable success signals such as:
  - three successful `indexed doc-...` lines
  - a `Top hits:` section
  - at least one non-empty hit result
  - no requirement to match exact scores or snippet text byte-for-byte

#### R-003 README must expose limits without sounding apologetic

Requirement:

- Add a section that explains the operational envelope and design limits plainly.
- The wording must be confident and direct.
- The wording must avoid implying unsupported platform ambitions.

Deliverables:

- "Current fit" section
- "Not supported yet" section
- "When to choose another system" section

Verification:

- a reader can determine whether their use case fits without scanning source code

### Theme 2: Trust Signals

#### R-004 Add CI for the default quality bar

Requirement:

- Add GitHub Actions workflow files under `.github/workflows/`.
- The Phase 1 baseline workflow must run:
  - `go test ./...`
  - `go vet ./...`
- The Phase 2 integration workflow must run:
  - `SIMPLYKB_DATABASE_URL=... go test ./... -run Integration`
  - against a ParadeDB service in CI
- Until the Phase 2 integration workflow lands, any release preparation or public-facing change that affects setup, migrations, or search behavior must run the integration command locally and record that verification in the pull request or release checklist.

Deliverables:

- baseline CI workflow file
- dedicated integration CI workflow file before `v0.1.0`
- visible badge in README

Verification:

- Phase 1: a new push and pull request trigger baseline automated checks
- Phase 2: the integration workflow passes in CI before `v0.1.0` is declared ready
- repository main branch shows passing or failing status clearly

#### R-005 Define release policy

Requirement:

- Document how releases are cut, named, and communicated.
- Use semantic intent even if the project is still `v0.x`.

Deliverables:

- `RELEASING.md`
- first version tag plan, such as `v0.1.0`
- statement of compatibility expectations

Verification:

- a contributor can understand what qualifies as patch, minor, and breaking change

#### R-006 Add a changelog strategy

Requirement:

- Create a changelog file or release-note convention.
- Keep the format simple and maintainable.

Deliverables:

- `CHANGELOG.md`
- release entry template

Verification:

- future changes can be summarized without guesswork

### Theme 3: Realistic Usage Confidence

#### R-007 Add at least one real embedder example

Requirement:

- Add one example that uses a real embedding provider or a clean provider adapter stub.
- The example must not suggest `HashEmbedder` is appropriate for production.
- If a real provider example would require unsafe secret handling, unavoidable billing assumptions, or unstable third-party setup, the work must degrade to a provider adapter stub instead of stopping.
- Hard-fail only if neither a safe real provider example nor a clearly documented adapter stub can be added without expanding project scope.

Deliverables:

- example code under `examples/`
- documentation section explaining when to use it
- environment variable list for the example
- explicit note describing whether the shipped example is:
  - a real provider example
  - or an adapter stub with exact handoff boundaries

Verification:

- a user can see a realistic integration shape without designing it from scratch
- the execution path does not stop merely because a public repo should not embed live provider credentials

#### R-008 Separate demo-safe guidance from production guidance

Requirement:

- Audit [README.md](../../README.md), [examples/quickstart/main.go](../../examples/quickstart/main.go), and future examples to make sure "demo mode" and "production mode" are clearly distinguished.

Deliverables:

- labels such as "local smoke/demo" and "production usage"
- one short section named "Production Notes"

Verification:

- a reader cannot reasonably mistake the hash embedder for production best practice

#### R-009 Add troubleshooting guidance

Requirement:

- Create a troubleshooting section or dedicated document.
- Cover the highest-probability failures first.

Required topics:

- database not reachable
- ParadeDB extension issues
- embedding dimension mismatch
- search returns no hits
- integration tests skipped

Deliverables:

- `docs/troubleshooting.md` or equivalent

Verification:

- common setup failures have documented next steps

### Theme 4: Contributor Experience

#### R-010 Add contributor guide

Requirement:

- Create `CONTRIBUTING.md`.
- Keep language simple and practical.

Required sections:

- repo purpose
- how to run tests
- when to add unit tests
- when to add integration tests
- how to keep scope narrow
- commit and PR expectations

Verification:

- a new contributor can make a small change without private instructions from the maintainer

#### R-011 Add issue and pull request templates

Requirement:

- Add GitHub templates under `.github/`.

Required templates:

- bug report
- feature request
- pull request template

Verification:

- incoming community contributions use consistent structure

#### R-012 Add architecture guardrails

Requirement:

- Extend docs to state which kinds of changes are usually welcome and which should be resisted.
- Tie this to the low-entropy design philosophy.

Deliverables:

- short "architecture guardrails" section in `CONTRIBUTING.md` or dedicated docs

Verification:

- contributors know that feature sprawl is not the default direction

### Theme 5: Quality And Public Safety

#### R-013 Public changes must preserve contract clarity

Requirement:

- Changes to request normalization, migrations, search behavior, and storage assumptions must be covered by tests.
- Known classes of drift must be prevented early.

Imported review findings that must remain guarded:

- migration must fail early on embedding dimension drift, anchored in `Store.Migrate` and `ensureEmbeddingDimensions` in `store.go`, and in `TestIntegrationMigrateRejectsEmbeddingDimensionDrift` in `integration_test.go`
- upsert must reject empty splitter output, anchored in `Store.UpsertDocument` in `store.go`, and in `TestIntegrationUpsertRejectsEmptySplitterOutput` in `integration_test.go`
- delete must normalize `documentID` consistently, anchored in `Store.DeleteDocument` in `store.go`, and in `TestIntegrationDeleteDocumentTrimsDocumentID` in `integration_test.go`
- schema must avoid redundant index noise, anchored in `schemaMigrations` in `schema.go`, and in `TestIntegrationMigrateDropsRedundantIndexes` in `integration_test.go`

Verification:

- existing tests continue covering these behaviors
- Phase 1 baseline CI must fail if unit or non-integration regression coverage breaks
- until the Phase 2 integration workflow lands, pull requests that affect setup, migrations, or retrieval contracts must record a successful local integration test run
- before `v0.1.0`, the dedicated integration workflow must pass in CI

#### R-014 Add compatibility notes

Requirement:

- Document tested and expected versions.

Minimum matrix:

- Go version
- ParadeDB image or baseline version
- local Docker requirement
- supported operating assumptions

Deliverables:

- compatibility section in README or dedicated docs page

Verification:

- setup ambiguity is reduced for external users

#### R-015 Define "done" for public-facing changes

Requirement:

- Any change that affects onboarding, setup, or contracts must update docs and verification steps.

Deliverables:

- lightweight checklist added to `CONTRIBUTING.md` or PR template

Required checklist items:

- code updated
- tests updated
- docs updated
- example impact checked
- release note impact checked

Verification:

- public-facing behavior does not drift away from public-facing docs

### Theme 6: Release Identity

#### R-016 Create first release milestone

Requirement:

- Define what `v0.1.0` means.

Suggested meaning:

- stable basic schema flow
- stable public API surface for four core methods
- tested local quickstart
- documented limits
- baseline CI in place
- dedicated integration CI in place

Deliverables:

- milestone checklist
- release note draft

Verification:

- maintainers can answer "what is officially stable right now?"

#### R-017 Add repository metadata polish

Requirement:

- Fill the public repository with basic polish signals.

Deliverables:

- repository description
- topics/tags
- license visibility
- README badges
- discussion of support expectations

Verification:

- GitHub page looks maintained and intentional

## Implementation Plan

This section turns requirements into work packages.

### Phase 1: Must-Have Public Foundation

Goal:

- make the repository feel safe and understandable for first-time visitors

Tasks:

1. Improve README top sections
2. Add first-run success path
3. Add CI workflow
4. Add `CONTRIBUTING.md`
5. Add PR and issue templates
6. Add compatibility and troubleshooting notes

Exit criteria:

- landing page clarity is strong
- CI exists
- contributors have clear instructions

### Phase 2: Adoption Confidence

Goal:

- make practical builders trust that the project is usable beyond a toy demo

Tasks:

1. Add real provider example
2. Add production notes
3. Add release policy
4. Add changelog
5. Add dedicated integration CI workflow
6. Define `v0.1.0` milestone

Exit criteria:

- README no longer relies mainly on `HashEmbedder`
- versioning and release expectations are documented
- integration CI is in place before `v0.1.0`

### Phase 3: Public Maturity

Goal:

- make the repository pleasant to evaluate and sustainable to maintain

Tasks:

1. Add repository badges and metadata polish
2. Expand troubleshooting based on real user questions
3. Add architecture guardrails and contribution boundaries
4. Review docs for consistency and duplication

Exit criteria:

- repository looks intentional, alive, and coherent

## Requirement To Implementation Traceability Matrix

Each requirement below maps to at least one implementation step, one target file area, and one verification action.

| Requirement | Implementation step(s) | Target files/modules | Verification anchor |
| --- | --- | --- | --- |
| R-001 | Phase 1, Step 1 | `README.md` | README cold-read check |
| R-002 | Phase 1, Step 2 | `README.md`, `examples/quickstart/main.go` | quickstart success signals |
| R-003 | Phase 1, Steps 1-2 | `README.md` | limits identifiable without code reading |
| R-004 | Phase 1, Step 3; Phase 2, Step 5 | `.github/workflows/ci.yml`, `.github/workflows/integration.yml`, `README.md` | baseline CI plus integration CI before `v0.1.0` |
| R-005 | Phase 2, Step 3 | `RELEASING.md` | release policy review |
| R-006 | Phase 2, Step 4 | `CHANGELOG.md` | initial changelog entry |
| R-007 | Phase 2, Step 1 | `examples/`, `README.md`, example docs | real provider example or adapter stub review |
| R-008 | Phase 2, Steps 1-2 | `README.md`, `examples/quickstart/main.go`, provider example docs | demo vs production distinction check |
| R-009 | Phase 1, Step 6; Phase 3, Step 2 | `docs/troubleshooting.md`, `README.md` | troubleshooting coverage review |
| R-010 | Phase 1, Step 4 | `CONTRIBUTING.md` | new contributor dry-run |
| R-011 | Phase 1, Step 5 | `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md` | GitHub template appearance |
| R-012 | Phase 3, Step 3 | `CONTRIBUTING.md` or dedicated architecture doc | guardrail review |
| R-013 | Phase 1, Step 3; Phase 2, Step 5; ongoing code/test updates | `store.go`, `schema.go`, `integration_test.go`, `.github/workflows/ci.yml`, `.github/workflows/integration.yml` | contract tests plus CI gates |
| R-014 | Phase 1, Step 6 | `README.md` or dedicated compatibility doc | version and environment review |
| R-015 | Phase 1, Steps 4-5 | `CONTRIBUTING.md`, PR template | public-change checklist audit |
| R-016 | Phase 2, Step 6 | `RELEASING.md`, `CHANGELOG.md`, milestone docs | `v0.1.0` milestone review |
| R-017 | Phase 3, Step 1 | repository settings, `README.md` | public GitHub page review |

## Verification Matrix

Each requirement must map to a verification action.

| Requirement | Verification |
| --- | --- |
| R-001 | README top section reviewed by someone unfamiliar with repo |
| R-002 | first-run steps executed on a clean environment |
| R-003 | limits can be identified without reading code |
| R-004 | baseline CI runs on push/PR in Phase 1, and integration CI passes before `v0.1.0` |
| R-005 | `RELEASING.md` reviewed and accepted |
| R-006 | `CHANGELOG.md` added with initial entry |
| R-007 | real provider example runs or adapter stub is documented with exact boundaries and fallback rationale |
| R-008 | docs clearly distinguish demo and production |
| R-009 | troubleshooting doc covers top five setup failures |
| R-010 | external contributor can follow guide without private help |
| R-011 | GitHub templates appear during issue/PR creation |
| R-012 | architecture guardrails written and visible |
| R-013 | baseline CI passes in Phase 1, local integration verification is recorded until dedicated integration CI lands, and integration CI passes before `v0.1.0` |
| R-014 | compatibility notes are explicit |
| R-015 | public-change checklist is adopted |
| R-016 | `v0.1.0` milestone definition is documented |
| R-017 | repo metadata and badges are visible on GitHub |

## Risk Register

### Risk 1: Over-documenting before release discipline exists

Problem:

- docs can become stale if process is weak

Mitigation:

- add small docs first
- tie docs updates to PR checklist

### Risk 2: Project starts sounding larger than it is

Problem:

- public docs may accidentally imply platform ambitions

Mitigation:

- repeat non-goals
- keep "when not to use this" visible

### Risk 3: Demo path is mistaken for production path

Problem:

- users may treat the hash embedder as a serious production option

Mitigation:

- clearly label demo-only paths
- add real provider example

### Risk 4: Public contributors push the project toward sprawl

Problem:

- well-meant contributions can increase entropy

Mitigation:

- add architecture guardrails
- state preferred direction early

### Risk 5: Quality bar becomes inconsistent

Problem:

- code, docs, and examples can drift apart

Mitigation:

- CI
- public-change checklist
- release notes discipline

## Recommended File Plan

The following file additions or updates are recommended.

### Update Existing

- [README.md](../../README.md)

### Add New

- `../../.github/workflows/ci.yml`
- `../../.github/ISSUE_TEMPLATE/bug_report.md`
- `../../.github/ISSUE_TEMPLATE/feature_request.md`
- `../../.github/pull_request_template.md`
- `../../CONTRIBUTING.md`
- `../../RELEASING.md`
- `../../CHANGELOG.md`
- `../troubleshooting.md`
- `../../examples/<provider-example>/...`

## Prioritized TODO List

If work must be sequenced tightly, use this order:

1. Improve README landing section
2. Add CI
3. Add `CONTRIBUTING.md`
4. Add troubleshooting and compatibility notes
5. Add PR and issue templates
6. Add release policy and changelog
7. Add dedicated integration workflow before `v0.1.0`
8. Add one realistic provider example or, if needed, a provider adapter stub with explicit boundaries
9. Prepare `v0.1.0`
10. Polish repo metadata and badges

## Kickoff Instruction

Use the following instruction when starting a new implementation conversation:

```text
You are implementing the public-readiness plan for simplykb.

Objective:
Turn the repository into an excellent out-of-the-box public open source project without expanding product scope.

Primary constraints:
- Preserve the low-entropy design philosophy.
- Do not add broad product features.
- Prefer documentation, trust signals, contributor guidance, and realistic examples.
- Keep language clear for first-time users.
- Keep demo guidance and production guidance clearly separated.

Required source documents:
- docs/plans/2026-04-10-simplykb-design.md
- docs/plans/2026-04-10-simplykb-open-source-readiness-design.md

Must-preserve quality guards:
- migration must fail early on embedding dimension drift
- upsert must reject empty splitter output
- delete must normalize documentID consistently
- schema must avoid redundant index noise

Execution order:
1. Update README landing and quickstart sections.
2. Add GitHub Actions CI.
3. Add CONTRIBUTING.md.
4. Add troubleshooting and compatibility documentation.
5. Add issue and pull request templates.
6. Add RELEASING.md and CHANGELOG.md.
7. Add dedicated integration workflow before `v0.1.0`.
8. Add one realistic provider example or, if needed, a provider adapter stub with explicit boundaries.
9. Review docs for consistency.

Verification requirements:
- Run go test ./...
- Run go vet ./...
- Run the documented quickstart path if files affecting it changed
- Confirm docs and examples match actual behavior

Stop conditions:
- Stop and report if the work requires changing product scope significantly.
- If a real provider example would require unsafe secrets or unstable billing assumptions, degrade to a provider adapter stub and continue.
- Stop and report only if neither a safe real provider example nor a clearly bounded adapter stub can be delivered without changing project scope.

Required final output:
- short summary of what changed
- verification commands run
- any remaining gaps against the plan
```

## Readiness Gate Report

### Completeness Gate: PASS

All required major sections are included:

- goals
- scope
- options
- detailed requirements
- implementation phases
- verification matrix
- risks
- kickoff instruction

### Traceability Gate: PASS

Each `R-*` requirement maps to implementation steps, target files or modules, and verification actions.

### Executability Gate: PASS

The plan is organized into ordered phases, explicit step mappings, and phased CI policy.

### Ambiguity Gate: PASS

The document avoids placeholders such as `TBD`, `etc`, or "fix later".

### Safety Gate: PASS

The plan protects project scope by explicitly stating non-goals, guardrails, and stop conditions.

## Final Recommendation

Do not try to make `simplykb` look huge.

Make it look:

- clear
- fast to try
- honest about its limits
- safe to trust
- easy to contribute to

That is the correct version of "excellent" for this project.
