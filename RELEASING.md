# Releasing simplykb

This project is a small embedded SDK.
Releases should optimize for trust, repeatability, and low surprise.

## Versioning Policy

Until `v1.0.0`, treat the project as pre-stable:

- use semver tags such as `v0.1.0`
- document breaking changes clearly in release notes
- avoid silent behavior changes in setup, migrations, or retrieval

After `v1.0.0`, breaking public API or behavior changes should require a major version bump.

## Release Checklist

Before cutting a release:

1. Confirm `go.mod` still declares `module github.com/LyleLiu666/simplykb`.
2. Run `make verify`.
3. If the release is a stable major release such as `v1.0.0`, also clear [docs/stable-major-release-readiness.md](docs/stable-major-release-readiness.md).
4. If schema, retrieval, or setup behavior changed, also run `make integration-benchmark` and record any meaningful regression or improvement in the release notes or PR summary.
5. Review `README.md`, `docs/troubleshooting.md`, and `examples/` for drift.
6. Update [CHANGELOG.md](CHANGELOG.md).
7. Push the release commit to a remote branch.
8. Verify from a fresh external Go module using the pushed commit SHA or a temporary release-candidate tag:

```bash
go mod init example.com/simplykb-check
go get github.com/LyleLiu666/simplykb@<commit-or-candidate-tag>
go mod tidy
```

9. Create the final version tag only after the fresh external module check succeeds.

If the fresh external module cannot fetch and build the pushed candidate, do not publish the final release tag.

## Release Notes Template

Use release notes that answer these questions quickly:

- What changed?
- Does setup change?
- Do migrations change?
- Does search behavior change?
- Do examples or docs change?
- Is there any breaking change?

## Compatibility Expectations

Releases should preserve explicit compatibility expectations instead of making users infer them.

At release time, confirm the published docs still state:

- the supported Go baseline
- the expected ParadeDB baseline or image
- the local Docker expectation
- the supported operating assumptions for local development

If any of those expectations change, call the change out in both the release notes and the compatibility section of `README.md`.

`make verify` now includes the runtime diagnostics path (`make doctor`) in addition to smoke and integration coverage, so release candidates should not skip it.

## Public Promise

The release promise for `simplykb` is not "many features fast".
It is:

- stable SDK shape
- predictable local setup
- explicit production boundaries
- low-entropy evolution

## Historical v0.1.0 Milestone

This section is kept as release history only.
Do not use it as the active release checklist for later tags.
Current releases use the checklist above, plus [docs/stable-major-release-readiness.md](docs/stable-major-release-readiness.md) for a future stable major release.

`v0.1.0` meant:

- the basic schema flow is stable
- upgrade regression coverage exists for older schema states
- the public SDK surface around `New`, `Migrate`, `UpsertDocument`, and `Search` is intentionally small and documented
- the quickstart works on a normal developer machine
- runtime diagnostics exist for operators and evaluators
- the project limits are documented clearly
- baseline CI is in place
- dedicated integration CI is in place
