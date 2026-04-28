# simplykb Documentation

This branch keeps the current reading path short.
Start with the current product contract, then move into the task-specific docs you need.
Historical plans live under [plans/archive/README.md](./plans/archive/README.md) so old design notes do not quietly masquerade as current guidance.
Release validation evidence lives under [release-validation/README.md](./release-validation/README.md) so one-time command records stay separate from current release rules.

## Reading Path

1. [README.md](../README.md)
   Role: `canonical`
   Use it for the public project shape, quickstart, supported boundaries, and operational commands.

2. [troubleshooting.md](./troubleshooting.md)
   Role: `supporting`
   Use it when local setup, migration state, or retrieval behavior looks wrong.

3. [CONTRIBUTING.md](../CONTRIBUTING.md)
   Role: `canonical`
   Use it for local development setup, test expectations, and contribution guardrails.

4. [RELEASING.md](../RELEASING.md)
   Role: `canonical`
   Use it when preparing or reviewing a release.

5. [stable-major-release-readiness.md](./stable-major-release-readiness.md)
   Role: `canonical`
   Use it for the extra gates that must pass before a stable major release such as `v1.0.0`.

6. [release-validation/README.md](./release-validation/README.md)
   Role: `historical index`
   Use it when you need release-gate evidence from a past validation run.

## Historical Design Material

- [plans/archive/README.md](./plans/archive/README.md)
  Role: `index`
  Use it only when you need design history, archived execution notes, or rationale that is already reflected in the current docs and code.

## Scope Note

The GitHub issue and pull request templates under `.github/` are workflow templates rather than part of the main reader-facing documentation tree.
