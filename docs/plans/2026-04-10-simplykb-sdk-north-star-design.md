# simplykb SDK North Star

## One-Sentence Goal

`simplykb` should feel like a small Go SDK that a developer can embed into an existing service, not like a separate search platform they must adopt.

## Product Reading Order

When a new developer lands on the project, they should understand these points in this order:

1. This is an embedded Go SDK.
2. Docker exists only to start the local ParadeDB dependency.
3. The main job is text chunking, indexing, and hybrid recall.
4. The integration path is `Migrate`, `UpsertDocument`, and `Search`.
5. If their use case needs a platform, this project is intentionally the wrong tool.

If the docs fail to communicate that order quickly, the project will look bigger, heavier, and more confusing than it really is.

## North-Star Developer Experience

The target experience for a practical Go developer is:

1. They find the repository and understand the scope in under 2 minutes.
2. They start a correct local ParadeDB in one command.
3. They run a quickstart and see a successful search result in under 10 minutes.
4. They copy a short code sample into their own service.
5. They replace `HashEmbedder` with a real embedding provider when moving beyond local evaluation.

The feeling should be:

"I added a library and a database dependency."

The feeling should not be:

"I adopted a new product with its own service boundary, control plane, and workflow model."

## What Docker Means In This Project

Docker is a delivery aid for local setup.
It is not the center of the product.

Why it exists:

- ParadeDB is required for the intended retrieval shape
- Docker is the easiest predictable way to get ParadeDB locally
- local setup must be cheap enough that developers will actually try the SDK

What it should not imply:

- that `simplykb` is mainly a Docker app
- that users are expected to call a standalone `simplykb` service
- that the main integration model is container-to-container instead of Go-to-library

## Hard Requirements For The SDK Positioning

The SDK story is only believable if all of these are true:

1. A fresh Go project can add the module successfully.
2. The README leads with the SDK story before the infrastructure story.
3. The default quickstart path is reliable on a normal developer machine.
4. A custom local port still works without hidden extra steps.
5. A new developer can tell what success looks like from the docs alone.
6. The repo shows where demo-only pieces end and production responsibilities begin.

If any one of these fails, the project will feel heavier than the intended north star.

## Success Criteria

The project can claim "easy to embed" only when these checks pass:

- `go get` works from a fresh external Go module
- `make smoke` works on the default port
- the custom-port smoke path is documented with the exact working command
- a fresh sample app can run `Migrate`, `UpsertDocument`, and `Search`
- the README explains why Docker is present without making Docker the product
- the README clearly states both best-fit and not-a-fit scenarios
- error messages fail early when the caller points to plain Postgres instead of ParadeDB

## Non-Goals

To protect the SDK shape, these are explicitly out of scope unless the product definition changes:

- hosted platform positioning
- standalone search service as the primary interface
- workflow engine or ingestion platform
- dashboard-first product surface
- multi-tenant auth and ACL inside the core SDK
- broad provider-specific orchestration features

## Decision Filter

Future documentation and product changes should be judged with one simple question:

"Does this make `simplykb` easier to embed into an existing Go service?"

If the answer is yes, it likely fits.
If the answer is "it makes the project look more complete as a platform", that is usually drift.

## Immediate Documentation Implications

The public docs should do the following first:

- state "embedded Go SDK" in the opening lines
- explain Docker as a local dependency helper
- show the shortest successful evaluation path
- show the shortest embedded usage shape
- define the boundary between demo and production usage

Only after that should the docs expand into architecture, troubleshooting, and contribution details.
