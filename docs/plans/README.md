# simplykb Plans

## Purpose

This branch is the plan space for `simplykb`.
It separates the one active implementation plan from older design notes that now mainly explain history.

The current product contract does not live here.
For the current project shape, start with [README.md](../../README.md).

## Reading Path

### Active

- [2026-04-22-simplykb-hardening-design.md](./2026-04-22-simplykb-hardening-design.md)
  Role: `working`
  Why it stays active: it still defines the next unfinished hardening slice around write amplification, explicit rebuilds, search diagnostics, and query caching.

### Historical

- [archive/README.md](./archive/README.md)
  Role: `index`
  Why it exists: it keeps completed or superseded plan material reachable without leaving it in the active reading path.

## Role Calls

| Document | Role | Current evidence | Action | Rationale |
| --- | --- | --- | --- | --- |
| [2026-04-22-simplykb-hardening-design.md](./2026-04-22-simplykb-hardening-design.md) | working | current repo still uses always-reindex upserts and simple search flow | kept active | still guides the next implementation decision |
| [archive/2026-04-10-simplykb-design.md](./archive/2026-04-10-simplykb-design.md) | historical | core architecture and boundaries are now reflected in [README.md](../../README.md) and landed code | moved to archive | useful origin-story design, but not needed for the next step |
| [archive/2026-04-10-simplykb-sdk-north-star-design.md](./archive/2026-04-10-simplykb-sdk-north-star-design.md) | historical | SDK positioning is now stated directly in [README.md](../../README.md) | moved to archive | explains reasoning, not active execution |
| [archive/2026-04-10-simplykb-open-source-readiness-design.md](./archive/2026-04-10-simplykb-open-source-readiness-design.md) | historical | repo now has CI badges, issue templates, [CONTRIBUTING.md](../../CONTRIBUTING.md), [RELEASING.md](../../RELEASING.md), troubleshooting, and provider-shaped examples | moved to archive | most planned deliverables have landed |
| [archive/2026-04-21-external-feedback-comparison-plan.md](./archive/2026-04-21-external-feedback-comparison-plan.md) | historical | its output was distilled into the active hardening design | moved to archive | useful provenance, but not a current execution plan |

## Terminology

- `working`: an active plan that still decides the next implementation move
- `historical`: preserved context that explains how the current direction emerged
- `canonical`: the current source of truth for project behavior or public contract

Within this topic, the canonical sources are the root [README.md](../../README.md), current code, and tests, not the plan files themselves.

## Next

- Read [2026-04-22-simplykb-hardening-design.md](./2026-04-22-simplykb-hardening-design.md) for the active plan.
- Read [archive/README.md](./archive/README.md) only when you need design history.
