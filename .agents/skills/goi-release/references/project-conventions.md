# Goi release conventions

Use this reference when preparing a version, building binaries, drafting release notes, or publishing a Goi release.

## Repository facts

- Module: `github.com/NeverStopDreamingWang/goi/v2`
- Canonical version: `const version = "vX.Y.Z"` in `version.go`
- Release commit title: `发布 vX.Y.Z 版本`
- GitHub repository: `https://github.com/NeverStopDreamingWang/goi`
- Configured remotes commonly include `github` and `gitee`; never infer that both should be pushed.
- Local tags may lag behind release commits and hosted Releases. Determine the previous release from multiple signals.

## Release artifacts

The repository tracks both CLI binaries even though generic executable patterns appear in `.gitignore`:

- `cmd/goi.exe`: `windows/amd64`
- `cmd/goi`: `linux/amd64`

Use the host-specific script rather than duplicating build commands:

- `cmd/build.bat` on Windows
- `sh ./cmd/build.sh` on Linux

Each script cross-compiles both targets with `CGO_ENABLED=0`, `-trimpath`, and `-ldflags="-s -w"`.

## Release-note style

Follow the most recent applicable Release body. The established format groups changes by import path and uses concise Chinese action prefixes:

- `添加-` for new capability
- `调整-` for behavioral or API changes
- `优化-` for improvements
- `修复-` or `修正-` for defects and documentation corrections
- `同步-` for generated templates, binaries, or version alignment
- `重大调整-` for intentionally prominent incompatible changes

Use backticks for packages, methods, fields, commands, and filenames. Describe observable changes and compatibility implications rather than repeating commit titles verbatim.

Preferred shape:

```markdown
## github.com/NeverStopDreamingWang/goi

修复-`SomeAPI` 在具体条件下的错误行为。

优化-相关验证逻辑并补充回归测试。

## github.com/NeverStopDreamingWang/goi/cmd

同步-Windows 与 Linux 命令行构建产物版本为 `vX.Y.Z`。

**Full Changelog**: https://github.com/NeverStopDreamingWang/goi/compare/vOLD...vNEW
```

Use only sections that have actual changes. Avoid generic headings such as “What's Changed” when the established package-oriented format is clearer. Keep the comparison versions exact and do not link to a comparison whose target tag has not been or will not be created.

## Release readiness

A locally prepared release should normally contain:

- the intended source and documentation changes;
- `version.go` updated to the target version;
- rebuilt `cmd/goi.exe` and `cmd/goi`;
- focused test evidence and `go build ./...` evidence;
- a clean `git diff --check` result;
- a release-note draft derived from the actual release range.

Do not hide environment-dependent integration-test failures. Record the failed command, cause, and narrower checks that passed.
