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
3. Review `README.md`, `docs/troubleshooting.md`, and `examples/` for drift.
4. Update [CHANGELOG.md](CHANGELOG.md).
5. Push the release commit to a remote branch.
6. Verify from a fresh external Go module using the pushed commit SHA or a temporary release-candidate tag:

```bash
go mod init example.com/simplykb-check
go get github.com/LyleLiu666/simplykb@<commit-or-candidate-tag>
go mod tidy
```

7. Create the final version tag such as `v0.1.0` only after the fresh external module check succeeds.

If the fresh external module cannot fetch and build the pushed candidate, do not publish the final release tag.

## Release Notes Template

Use release notes that answer these questions quickly:

- What changed?
- Does setup change?
- Do migrations change?
- Does search behavior change?
- Do examples or docs change?
- Is there any breaking change?

## Public Promise

The release promise for `simplykb` is not "many features fast".
It is:

- stable SDK shape
- predictable local setup
- explicit production boundaries
- low-entropy evolution
