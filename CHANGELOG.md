# Changelog

## 0.7.0 — 2026-02-21

- **New `bw start` command** — claims and begins work on an issue in one step. Validates the issue is open and unblocked, moves it to `in_progress`, and assigns from `git config user.name` (or `--assignee`). If the issue has unresolved blockers, it shows the blocker titles and suggests `bw ready`. The `--assignee` flag has been removed from `bw create` — assignment now means active work, not planning.
- **Actionable tips in `bw show`** — dependency sections now walk the blocker tree to surface only the leaf issues that need work. If A←B←C←D, showing A displays only D. New BLOCKED BY and UNBLOCKS sections replace the old flat dependency lists. Use `--only` to filter output to specific sections (replaces `--short`).
- **`bw graph` removed** — its job is now done better by show's dependency sections.
- **`bw close` strikethroughs the title** for a visual "done" cue.
- **`bw prime` injects project state inline** — ready queue and issue counts now appear in context under "Finding and Doing Work" rather than appended at the end.
- **Reworked `bw prime` and `bw onboard` prompts** — prime now emphasizes landing the work (commit, close, sync) and connects issues to commits. The onboard prompt has also changed — if you've customized your agent configuration based on the old output, re-run `bw onboard` to pick up the new version.
- **Sync uses system git for network operations** — fixes SSH failures on macOS when the SSH agent has no loaded keys.

## 0.6.2 — 2026-02-20

- `bw show` wraps long text to terminal width with indent-aware formatting
- Closing an issue now reveals any newly-unblocked issues that were waiting on it

## 0.6.1 — 2026-02-20

- Dotted child IDs: creating a child issue automatically generates a hierarchical ID (e.g., `abc.1`)
- `bw list` now shows blocking/blocked-by relationships inline
- `bw create --silent` suppresses output (useful for scripting)
- Improved `bw prime` template with styled output and agent-id callout

## 0.6.0 — 2026-02-20

- `bw delete` command with preview mode and `--force` to confirm
- `bw comments` subcommand to list and add comments on issues
- `bw history` command to show commit history for an issue
- `--parent` flag on `create` and `update` with cycle detection
- TTY-aware colored output (IDs, priorities, section headers, success states)
- `bw upgrade` now shows download progress, step narration, and changelog for the version span
- `bw list` defaults to open + in\_progress; new `--grep` filter for searching title and description
- `install.sh` for curl-based installation

## 0.5.3 — 2026-02-20

- `bw import -` reads from stdin, enabling piping (`bd export | bw import -`)
- Dependency injection for upgrade command enables deterministic testing (84% coverage)
- Increased test coverage for init and blocked commands

## 0.5.2 — 2026-02-20

- Issue IDs now use base36 (`0-9a-z`) with adaptive length starting at 3 characters, growing as needed to avoid collisions
- `id.retries` config option to tune retries per length (default 10)
- Increased test coverage for defer, undefer, upgrade, and sync commands

## 0.5.1 — 2026-02-20

- `bw show` parity: multi-ID support, `--short` flag, `view` alias, rich dependency display with titles and priority
- New `updated_at`, `closed_at`, and `close_reason` fields tracked across all issue mutations
- `bw create --labels/-l frontend,urgent` to set labels at creation time
- Export/import now round-trips `updated_at`, `closed_at`, and `close_reason`

## 0.5.0 — 2026-02-20

- **Breaking:** Priority scale shifted from 1-5 to 0-4 (P0=highest, default P2); existing repos require `bw upgrade repo`
- **Breaking:** `bw create` now requires quoted multi-word titles (no more implicit joining of bare words)
- Repo schema versioning and migration system; `bw upgrade repo` to apply format upgrades
- Unknown CLI flags are now rejected instead of silently ignored
- New `bw blocked` command to list issues waiting on dependencies
- `bw ready` excludes in-progress issues and supports deferred issues
- Lossless export/import round-trip with full integration test coverage

## 0.4.3 — 2026-02-20

- Richer per-command help: descriptions, usage examples, and improved layout
- `--help` now writes to stdout (was stderr) and exits 0

## 0.4.2 — 2026-02-20

- Per-command `--help` with flag/positional descriptions, auto-generated from command metadata
- Replace git CLI + worktree with go-git for atomic internal operations

## 0.4.1 — 2026-02-19

- Fix `bw graph --all` incorrectly filtering out closed issues that have dependency links
- CLI integration tests now contribute to coverage via instrumented binary (Codecov: 69% → 77%)
- Add beads/beadwork migration guide to README

## 0.4.0 — 2026-02-19

- Refactor all command handlers for unit testability: return `error` + write to `io.Writer` instead of `fatal()`/`os.Stdout`
- Comprehensive test coverage across all packages (74.8% → 78.4%)
- Fix `os.Args` bug in close, create, label, and update commands (now correctly use `args` parameter)

## 0.3.3 — 2026-02-19

- Codecov integration for coverage tracking and badges
- Grouped commands by topic in help output and README

## 0.3.2 — 2026-02-19

- `bw init --force` to reinitialize (destroys and recreates the beadwork branch)
- Prefix validation: alphanumeric, hyphens, underscores, max 16 chars
- Derived prefix now preserves casing, truncates to 8 chars, strips invalid characters

## 0.3.1 — 2026-02-19

- `bw upgrade` command: check for and install newer versions from GitHub releases
- Automated release binaries via GoReleaser (linux/darwin/windows, amd64/arm64)

## 0.3.0 — 2026-02-19

- Fix `bw init` in projects with pre-commit hooks (skip hooks for internal beadwork commits)
- Fix noop updates erroring when no file changes result (e.g., setting an already-set status)
- Default `bw list` to open issues with limit of 10

## 0.2.0 — 2026-02-19

- Config system: `.bwconfig` with `bw config get|set|list` and `default.priority` support
- Import/export: `bw export` and `bw import` for beads JSONL interop (collision detection, `--dry-run`, dependency wiring)
- Restyled `show`, `list`, and `ready` output with status icons and priority dots
- CLI integration test suite (27 tests covering command output)
- Intent replay test suite (20 tests covering all replay verbs and edge cases)

## 0.1.0 — 2026-02-19

Initial release.

- Filesystem-native issue tracking on a git orphan branch
- Full CLI: create, show, list, update, close, reopen, ready, label, link, unlink, graph
- Dependency tracking with `bw ready` for unblocked work
- Intent-replay sync (`bw sync`) with automatic conflict resolution
- Agent integration: `bw onboard` and `bw prime`
- `--json` output on show, list, ready, and graph
- CI via GitHub Actions
