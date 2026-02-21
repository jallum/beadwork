# Changelog

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
