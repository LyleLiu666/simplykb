# simplykb Plans

## Purpose

This branch is now a narrow index for plan history and any future active plan.
The current product contract does not live here.
For the current project shape, start with [README.md](../../README.md).

## Current State

There is no active implementation plan in this branch as of `2026-04-23`.

The retrieval hardening slice that previously lived here is already reflected in:

- [README.md](../../README.md)
- [docs/troubleshooting.md](../troubleshooting.md)
- [CHANGELOG.md](../../CHANGELOG.md)
- current code and tests

That design note is still preserved, but it now lives in [archive/README.md](./archive/README.md) as historical material.

## Reading Path

- For current behavior and supported boundaries, read [README.md](../../README.md).
- For setup and runtime failures, read [docs/troubleshooting.md](../troubleshooting.md).
- For contributor workflow, read [CONTRIBUTING.md](../../CONTRIBUTING.md).
- For release workflow, read [RELEASING.md](../../RELEASING.md).
- For design history, read [archive/README.md](./archive/README.md).

## Role Calls

| Document | Role | Current evidence | Action | Rationale |
| --- | --- | --- | --- | --- |
| [archive/2026-04-22-simplykb-hardening-design.md](./archive/2026-04-22-simplykb-hardening-design.md) | historical | `ReindexDocument`, `SearchDetailed`, query-cache configuration, diagnostics, benchmarks, and doc updates are now present in the repo and called out in [CHANGELOG.md](../../CHANGELOG.md) for `2026-04-23` | moved to archive | preserves design rationale, but no longer decides the next implementation move |
| [archive/README.md](./archive/README.md) | index | keeps completed plans reachable from one place | kept active | readers can find history without treating it as current guidance |

## Rule For Future Plans

Keep at most one active implementation plan in this directory.
Once its deciding changes have landed and the current contract is documented elsewhere, move it into `archive/`.
