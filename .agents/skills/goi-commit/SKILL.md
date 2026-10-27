---
name: goi-commit
description: Review, stage, and commit focused changes in the Goi repository when asked for a commit message, staging plan, or Git commit. Do not use for version-release commits, history rewriting, or pushing changes.
---

# Goi Commit

Create focused, reviewable commits that match this repository's established Chinese commit-message style without absorbing unrelated worktree changes.

## Determine the requested outcome

Distinguish between:

- **Message only:** inspect the diff and return a proposed commit title and optional body. Do not stage or commit.
- **Stage changes:** stage only the reviewed paths requested or clearly included in the task. Do not commit unless requested.
- **Create a commit:** review, verify, stage the intended paths, and create one or more commits when the user explicitly asks to commit.

A request to draft a message does not authorize staging. A request to commit does not authorize amending, rebasing, pushing, tagging, or publishing. Use the project `$goi-release` workflow for version bumps and commits named `发布 vX.Y.Z 版本`.

## Inspect before staging

Always inspect:

1. `git status --short`
2. unstaged and staged diffs
3. untracked files that might be included
4. recent commit subjects, normally the latest 15–25 commits

Separate pre-existing user changes from changes made for the current task. Preserve unrelated edits and existing staged state. If already-staged files cannot be confidently attributed to the requested commit, stop and ask before changing the index or committing them.

Review generated files, dependency files, binaries, and secrets deliberately. In this repository:

- `cmd/dist/` contains ignored local build artifacts. Never stage or commit them; explicitly requested release assets are handled separately by the release workflow.
- Include `go.sum` only when its dependency changes are understood and belong to the commit.
- Project skills under `.agents/skills/` are normal source files when their creation or update is the requested change.

## Form logical commits

Prefer one commit for one coherent change. If the worktree contains independent changes, propose or create separate commits only when each commit remains buildable and understandable on its own.

Do not split implementation from required tests or synchronized internationalization resources. Do not combine unrelated fixes merely because they are currently dirty.

Before committing Go changes, run `gofmt` on the changed Go files and verification proportional to the change. Prefer focused tests plus `go build ./...`; run broader tests when practical. Also run `git diff --check`. If relevant verification fails, report the failure and do not commit unless the user explicitly instructs otherwise with awareness of the failure.

## Match the repository message style

Use a concise Chinese subject based on the actual diff. Common prefixes are:

- `添加-` for a new capability or file
- `修复-` for a functional defect
- `修正-` for documentation, naming, or small correctness fixes
- `优化-` for a behavior-preserving improvement
- `调整-` for an API or behavior change
- `重构：` for broad structural or naming work

Use backticks around package names, files, methods, fields, and commands when it improves precision. Prefer a single clear subject such as:

```text
修复-`db` 验证器兼容命名字符串类型
```

Avoid appending an entire bullet list to the subject. Add an optional body when multiple related details, compatibility implications, or verification context materially help. The body should explain what changed and why rather than repeat the title.

Reserve `发布 vX.Y.Z 版本` for the release workflow.

## Stage and commit safely

- Prefer path-specific staging such as `git add -- <reviewed-paths>`.
- Do not use `git add .`, `git add -A`, or broad globbing unless the user explicitly placed the entire reviewed worktree in scope.
- After staging, inspect `git diff --cached --stat` and `git diff --cached` before committing.
- Use non-interactive commit commands with the approved subject and body.
- Never use `--amend`, force operations, or history rewriting unless explicitly requested.
- If a commit hook fails or modifies files, inspect the resulting status before retrying. Do not bypass hooks unless explicitly authorized.

## Report the result

For a proposed message, return the exact title and optional body in a copyable block.

For an actual commit, report:

- commit hash and subject;
- files included;
- verification performed and any limitations;
- remaining staged, unstaged, and untracked changes.

Do not claim the worktree is clean without checking it after the commit. Do not push unless separately requested.
