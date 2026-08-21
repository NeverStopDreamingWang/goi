---
name: goi-release
description: Prepare Goi framework releases when asked to bump a version, draft release notes, build release binaries, check release readiness, create a tag, or publish a GitHub or Gitee release. Do not use for ordinary feature work or unrelated Go builds.
---

# Goi Release

Prepare a reproducible release while preserving the repository's established versioning, build artifacts, and Chinese release-note style.

Read [references/project-conventions.md](references/project-conventions.md) before changing a version, building release artifacts, drafting final release notes, or publishing a release.

## Establish the requested scope

Distinguish these outcomes:

- **Notes only:** inspect history and return a release-note draft without changing files.
- **Prepare locally:** update the version and release artifacts, run checks, and report the resulting diff. Do not commit, tag, push, or create a remote release unless requested.
- **Publish:** perform only the explicitly requested commit, tag, push, asset upload, and release operations. Confirm which remote or hosting service is in scope; do not assume that both GitHub and Gitee should be mutated.

If the request is ambiguous, stop after the least-mutating outcome that still provides value. Never overwrite an existing tag or release.

## Inspect release context

Before making changes:

1. Confirm the repository root and module path.
2. Read `version.go`, `cmd/build.bat`, `cmd/build.sh`, and the relevant README build instructions.
3. Inspect `git status`, recent commits, current branch, remotes, and available tags.
4. Identify the previous published version. Treat local tags as corroborating evidence, not the sole source: this repository can contain release commits whose tags are not present locally. When needed and authorized, inspect the hosting service's Releases data; otherwise locate the latest `发布 vX.Y.Z 版本` commit and compare it with `version.go`.
5. Preserve unrelated dirty-worktree changes. Clearly separate pre-existing changes from release changes.

Base release notes on the exact commit range between the previous release commit or tag and the proposed release state. Do not invent changes from conversation context or an outdated README.

## Prepare the version

- Use semantic versions with a leading `v`, such as `v2.0.2`.
- Treat `version.go` as the canonical framework/CLI version source.
- Search for other hard-coded occurrences of the old version before editing, but change only occurrences that genuinely represent the product version.
- Follow the user's requested patch, minor, or major version. If the intended compatibility level is materially ambiguous, ask before selecting it.
- Do not edit generated binaries by hand.

## Verify and build

Run checks proportionate to the change, normally including:

1. Format changed Go files with `gofmt`.
2. Run focused tests for changed packages.
3. Run `go test ./...` when the environment supports the repository's integration tests.
4. Run `go build ./...`.
5. Run `git diff --check` and inspect the final status and diff summary.

Do not describe a failed test as pre-existing without evidence. If integration tests require unavailable database drivers or services, report the exact limitation and retain the focused test/build evidence; do not claim the full suite passed.

Build release binaries through the checked-in host script:

- Windows: `cmd\build.bat`
- Linux: `sh ./cmd/build.sh`

Both scripts must produce `cmd/goi.exe` for `windows/amd64` and `cmd/goi` for `linux/amd64`. Verify that both files were rebuilt and are included in the release diff. On the native platform, execute the matching binary to confirm it reports the target Goi version. Do not attempt to execute the cross-compiled binary on the wrong operating system.

## Draft release notes

Follow [references/project-conventions.md](references/project-conventions.md). Group changes by Go import path, use the project's Chinese action prefixes, identify user-visible compatibility implications, and end with the exact comparison link:

```text
**Full Changelog**: https://github.com/NeverStopDreamingWang/goi/compare/<previous>...<target>
```

Exclude working-tree changes that will not be part of the release. A draft for uncommitted work must be presented as a draft, not as an already published changelog.

## Commit, tag, and publish

Only perform these actions when the user explicitly requests them:

1. Recheck that tests/builds completed at the stated level and the release diff contains only intended files.
2. Use the established release commit title: `发布 vX.Y.Z 版本`.
3. Check local and target-remote tag existence before creating `vX.Y.Z`.
4. Push only the requested branch/tag to the requested remote.
5. Create the release with the approved notes and requested assets. Do not silently create, replace, or delete remote assets.
6. Return the release URL, tag, commit, uploaded assets, verification summary, and any checks that could not run.

Stop before publication if the target version already exists, the release diff includes unexplained changes, required checks fail, or remote state differs from the prepared assumptions.
