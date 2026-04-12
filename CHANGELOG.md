# Changelog

## 0.12.2 — 2026-04-12

- **Subtree blocker display fix** — `bw ready` epics with children no longer show `[blocked by: ...]` annotations for their own descendants. The subtree-aware classification added in 0.12.1 correctly identified these epics as ready, but the display layer still rendered internal blocker IDs, making ready epics look blocked. Internal blockers are now filtered from annotations alongside closed blockers.

## 0.12.1 — 2026-04-07

- **Subtree-aware `bw ready` and `bw blocked`** — epics with children are now classified based on their full subtree instead of individually. If all blockers are internal to the subtree, the parent shows as ready and descendants are suppressed from both lists. If any descendant has an external blocker, the parent shows as blocked with those external blocker IDs aggregated. This reduces noise for projects with deep issue hierarchies — you see the actionable root, not every leaf.

## 0.12.0 — 2026-04-02

- **Due dates** — new `--due` flag on `bw create` and `bw update` sets deadlines without changing issue status. `bw list --overdue` filters to past-due items. Overdue items float to the top within their priority band in `bw list` and `bw ready`, and show an `(OVERDUE since YYYY-MM-DD)` marker in `bw show`. Clear a due date with `bw update <id> --due ""`. Date-only values use end-of-day semantics: `due: "2027-04-15"` isn't overdue until April 16.

- **Time-granular dates** — `bw defer` and the `--due`/`--defer` flags now accept time expressions: `in 15 minutes`, `tomorrow at 2pm`, `3pm`, `14:00`, or full RFC3339. Sub-day values are stored as RFC3339 with your local timezone offset. Date-only expressions (`2027-06-01`, `next monday`) continue to produce plain `YYYY-MM-DD`.

- **Query-time deferral expiry** — deferred items whose `defer_until` has passed now reappear in `bw ready` and default `bw list` automatically — no poller or cron needed. On-disk status stays `deferred`; the expiry is evaluated at query time. `bw defer bw-abc "in 15 minutes"` just works. `bw prime` surfaces expired deferrals in a dedicated "Reminders" section. `bw list --deferred` still shows all deferred items regardless of expiry, and `bw undefer` still works for manual cleanup.

- **Export/import pass-through** — `defer_until` and `due` fields pass through as-is in export/import. No more `T00:00:00Z` padding on export or time stripping on import. Old exports with padded values import without error.

- **Sort order change** — issues now sort by priority ascending, then overdue-first within each priority band, then creation date ascending.

- **Agent guidance** — `bw prime` now includes due-date/deferral guidance, a date expression reference, overdue count, and a reminders section for expired deferrals.

- **Note**: Mixed-version usage (old binary reading new issue data) will silently drop the `due` field on write-back. Data is recoverable from git history.

## 0.11.8 — 2026-04-01

- **`-C <dir>` flag** — run `bw` as if started in `<dir>`, matching `git -C` semantics. The repo is discovered from the specified directory without changing the process working directory. Useful for scripting against a different repo or running `bw` from outside a checkout: `bw -C /path/to/repo list`.

- **`bw onboard` pipe-friendly output** — when stdout is not a TTY (e.g. `bw onboard | pbcopy` or `bw onboard >> CLAUDE.md`), only the raw snippet text is emitted, without headings, code fences, or the "How it works" section. TTY output is unchanged.

- Minor prime prompt wording adjustments.

## 0.11.7 — 2026-03-25

- **Cyclic dependency rejection** — `bw dep add` now rejects edges that would create circular dependencies (e.g. A blocks B, B blocks C, then C blocks A). Previously only self-loops were caught; transitive cycles were silently accepted, creating logically incoherent state where blocked issues could never become unblocked. The check uses a DFS walk of the reachable subgraph, consistent with the existing parent-child cycle detection.

## 0.11.6 — 2026-03-13

- **Compact dependency annotations** — issues that block (or are blocked by) multiple others now show a single grouped annotation instead of one per ID. For example, `[blocks: bw-a, bw-b]` instead of two separate `[blocks: bw-a]` `[blocks: bw-b]` tags. Reduces visual noise in `bw list` and `bw ready` output, especially for epics with many children.

- **`bw show` simplified to single ID** — `bw show` now accepts exactly one issue ID instead of multiple. Multi-ID usage was rare in practice and made JSON output awkward (array vs object). `bw show --json` now returns a plain JSON object instead of a single-element array, which is easier to pipe into `jq` and other tools.

## 0.11.5 — 2026-03-13

- **`bw ls` alias** — `ls` is now an alias for `bw list`, saving a few keystrokes for the most common query command.

- **`--parent` filter for `bw list`** — filter issues by parent ID to see only children of a given epic. Example: `bw list --parent bw-a3f8`. Useful when reviewing progress on a specific epic without noise from unrelated work.

## 0.11.4 — 2026-03-10

- **Bug fix: false "uncommitted changes" warning** — `bw prime` and `bw ready` could report uncommitted changes on a clean working tree. The dirty-tree check used go-git's in-process status, which disagrees with real git on worktree boundaries and file modes. Now uses `git diff --quiet HEAD`, which is both correct and faster on large repos.

- Updated go-git to v5.17.0.

## 0.11.3 — 2026-03-10

- **Bug fix: TTY line wrapping** — lines with dependency annotations (e.g. `[blocks: bw-xyz]`) wrapped prematurely because the wrap engine counted invisible color markers as visible width, overcounting each line by ~51 characters. Most visible on `bw ready` and `bw list` output with `blocks`/`blocked by` annotations.

## 0.11.2 — 2026-03-08

- **Git context in `bw prime` and `bw ready`** — output now shows the current branch, clean/dirty state, and last commit so agents know where they are without a separate `git status`.

- **`bw init` surfaces onboard instructions** — after initializing, `bw init` now tells you about `bw onboard` so new users discover agent integration setup immediately.

- **Standardized worktree branch naming** — the prime prompt now includes a naming convention (`<id>/<short-description>`, e.g. `bw-a1b/fix-auth-bug`) so agents produce consistent branch names.

- **Uncommitted changes warning** — `bw prime` and `bw ready` now warn when the working tree has uncommitted changes.

- **Styled open checkboxes** — TTY output uses `☐` (dimmed) for open checkboxes instead of `[ ]`, matching the `✓` styling for completed items.

- **Blocked status in `bw ready` legend** — the TTY legend now includes the blocked indicator.

- **Bug fix: `bw defer` relative dates** — inputs like "2 weeks" and "next monday" now parse correctly.

- **Bug fix: line wrapping** — bullet items and dependency annotations no longer break mid-word.

- **`--raw` renamed to `--x-raw`** — internal/debug flags now use an `--x-` prefix to distinguish them from stable flags.

## 0.11.1 — 2026-03-08

- **Styled upgrade changelog** — the release notes shown during `bw upgrade` now go through the same markdown rendering pipeline as `bw show` and `bw list`, gaining heading colorization, bold/code inline styling, and terminal-width wrapping.

## 0.11.0 — 2026-03-08

- **Markdown-native output** — `bw show`, `bw list`, `bw ready`, and `bw start` now emit semantic markdown with embedded tokens (`{status:open}`, `{id:bw-xyz}`, etc.) that resolve to styled ANSI on TTY, clean unicode in markdown mode, or raw tokens for debugging. This replaces the old Go template rendering pipeline. Use `--x-render-as tty|markdown|raw` to preview the different modes.

- **Rewritten prime prompt** — `bw prime` is shorter (~300 words, down from ~700) and structured around a delivery-awareness question: _"How should this land?"_ with three tiers — quick fix (no ticket), branch/PR (ticket + worktree), or multi-step (epic). The agent asks the user instead of guessing, which tested more reliably than blanket "every change gets a ticket" rules. Dirty worktree warnings now appear as a prominent `[!WARNING]` block at the top.

- **Agent-agnostic delegation** — the prime prompt no longer references Claude Code–specific tools like `isolation: "worktree"`. Instead it states consequences: _"Each delegated task needs its own worktree… Agents that can't request approvals can't land work — plan accordingly."_ Agents infer the right mechanism on their own. Tested 9/9 on both principle-based and explicit variants.

- **Agent detection** — new `internal/agent` package detects the invoking AI agent (Claude Code, Gemini CLI, Cursor) via environment variables, enabling agent-aware behavior in future commands.

- **Improved onboarding** — `bw onboard` now recommends adding the bootstrap prompt to `CLAUDE.md` (not `AGENTS.md`, which Claude Code doesn't read). References are agent-agnostic, mentioning `CLAUDE.md`, `GEMINI.md`, and `COPILOT.md`.

- **Bug fix: closed blockers no longer clutter output** — `bw list` dependency annotations now filter out already-closed blockers.

- **Repositioned as work management** — README reframed from "issue tracking" to "git-native work management." Added Discord community link.

- **Prompt design documentation** — new `docs/prompts/` directory with design requirements for each prompt (`prime.md`, `agents.md`, `start.md`) and a full experimentation methodology (`prompts.md`).

## 0.10.0 — 2026-03-06

- **Automatic upgrade notifications** — `bw` now checks for newer releases once per day (throttled via a local cache). When a new version is found, it creates a P1 upgrade bead with the changelog and instructions to run `bw upgrade`. The bead updates itself if further releases appear, and `bw upgrade` auto-closes it when complete. This runs silently in the background of any store-using command — no new flags or commands required.

- **`bw prime` now includes issue ID prefix** — the prime output resolves `{prefix}` placeholders so agents see the project's actual prefix (e.g., `bw-XYZ`) in the model description, reducing ambiguity about issue ID format.

- **Improved agent onboarding prompt** — the prime directive in `prompts/agents.md` now emphasizes that untracked/unlanded work is effectively void, better aligning with how agents actually lose context across sessions.

- **Project-level AGENTS.md** — replaced the symlink to `prompts/agents.md` with a standalone file describing beadwork's purpose, architecture, and repo structure for agents working on beadwork itself.

## 0.9.0 — 2026-02-28

- **`bw start` is now a full point-of-action briefing** — starting an issue prints type-aware guidance for both beginning and landing the work. Epics get "work through children via `bw ready`" while tasks and bugs get scoped commit steps. Children, dependencies, and comments are rendered inline from the template, so the briefing adapts to each issue type. Comments now always appear regardless of issue type (previously they were missing for non-standard types like `feature`).

- **`bw reopen` accepts in-progress issues** — you can now run `bw reopen` on an `in_progress` issue to reset it to `open` and clear the assignee, returning it to the ready queue without closing first. Useful when you need to unclaim work.

- **`bw ready` sorts by priority** — output is now ordered by priority (P0 first) then creation date, matching `bw list`. Previously the order was arbitrary.

- **`bw prime` shows work in progress** — the prime output now includes a "Work In Progress" section alongside ready work, so you can see what's already claimed at a glance.

- **`bw sync` no longer blocked by pre-push hooks** — the internal push for the beadwork data branch now skips user-configured hooks, which are irrelevant to the data branch and could block syncing.

- **`bw dep remove` errors on non-existent dependencies** — previously succeeded silently; now returns `"no dependency: A does not block B"`.

- **Fixed dependency display bugs** — `bw show` could display sibling issues in the BLOCKED BY section instead of only transitive blockers. `bw ready` and `bw list` showed closed blockers in annotations. Both are fixed.

- **Intent replay fixes** — `bw sync` conflict resolution now correctly replays `start`, `defer`, and `undefer` operations, preserves deterministic IDs during create replay, and encodes descriptions faithfully. These fixes improve reliability when syncing concurrent changes.

- **Improved prompts** — `bw prime` worktree guidance now names the scope-creep temptation ("that related fix is a new ticket") and embeds worktree creation into the "Starting the Work" workflow. `bw start` landing instructions include PR workflow guidance when applicable. Re-run `bw onboard` to pick up the new prompts.

- **Acceptance test harness** — golden-file acceptance tests verify end-to-end command sequences with deterministic state, including export/import round-trip fidelity.

## 0.8.1 — 2026-02-22

- **`bw show` header includes comment count** — the header bracket now shows how many comments an issue has (e.g., `· 3 comments`) or `· NO COMMENTS` when empty, so you can tell at a glance whether there's discussion without scrolling to the bottom.

## 0.8.0 — 2026-02-22

- **`bw start` now shows full issue context** — starting an issue prints the description, children, dependencies, comments, and a "Landing the Work" section with next steps. Delegates get the full picture without needing a separate `bw show`.

- **`bw ready` groups child issues under their parent** — epics and their children are visually grouped with indentation instead of appearing as a flat list. Standalone issues print first, followed by parent/child groups.

- **`bw show` displays children for epics** — a new CHILDREN section lists each child with status, priority, and inline dependency annotations. Use `--only children` to show just the children.

- **`bw comment` replaces `bw comments`** — simplified to a single verb: `bw comment <id> "text"`. The old `bw comments` still works as an alias. Listing comments has moved to `bw show` (which already displayed them).

- **`--dry-run` global flag** — run any command without committing changes to the beadwork branch. Shown in `bw --help` under Global Flags. Useful for previewing what a command would do.

- **Deterministic state for testing** — `bw create --id <ID>` pins the issue ID, and `BW_CLOCK=<RFC3339>` freezes timestamps. Together they make `bw export` output fully reproducible, enabling golden-file acceptance tests.

- **Faster startup** — removed `git` subprocess calls from the hot path (`findGitDir` and `gitUserName` replaced with pure-Go equivalents). Every `bw` invocation is faster, especially on macOS where subprocess startup carries SSH agent overhead.

- **Improved prompts** — `bw prime` and `bw onboard` now include guidance on delegating work to sub-agents, using worktrees, and closing out tickets after verification. If you've customized agent configuration, re-run `bw onboard` to pick up the new prompts.

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
