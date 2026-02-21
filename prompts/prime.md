# Beadwork

This project tracks work with `bw`. Run `bw prime` after compaction or new session.

## How It Works

Issues have **status** (open → in_progress → closed, or deferred), **priority** (P0 critical → P4 backlog, default P2), and optionally **dependencies**, **labels**, **comments**, and **parent** relationships.

`bw ready` is your starting point — it shows open issues with no unresolved blockers, sorted by priority. `bw show <id>` for details. `bw history <id>` to see what happened in previous sessions.

## Working

- Claim work with `bw update <id> --status in_progress` before starting.
- Record progress as you go — `bw comments add <id> "what happened"`. Context in your working memory can disappear without warning. Comments persist.
- If you discover new work mid-task, create an issue. Don't create issues for things you'll finish in the same breath.
- Before ending your session: commit code, push your branch, `bw close <id>` (with `--reason` if the resolution isn't obvious), and `bw sync`. Nothing should exist only in your context window.

## Commands

Every command supports `--help`. Read commands accept `--json`.

```bash
bw ready                               # Unblocked work
bw blocked                             # Issues waiting on dependencies
bw list                                # Open + in-progress issues (--all, --grep, --status, --label)
bw show <id>                           # Full detail with actionable blockers and comments
bw history <id>                        # Mutation log from git history
bw create "Title" -p 2 -t task         # New issue (P0-P4; task, bug, or epic)
bw update <id> --status in_progress    # Change fields (--title, --priority, --assignee, --parent)
bw close <id> [--reason "..."]         # Mark complete
bw comments add <id> "text"            # Leave a note for future sessions
bw dep add <blocker> blocks <blocked>  # Dependency link
bw label <id> +bug -wontfix            # Add/remove labels
bw sync                                # Fetch, merge, push
```

<!-- IF workflow.agents == multi -->

## Parallel Agents

Multiple agents need separate git worktrees to avoid file conflicts. `bw` itself is concurrent-safe — issue state is shared across all worktrees. Claim work (`--status in_progress` with `--assignee <agent-id>`) before starting so other agents skip it.

<!-- END -->
<!-- IF workflow.review == pr -->

## Code Review

Always work on a feature branch — never commit directly to main. Push and open a pull request when the work is ready. Reference the issue ID in the PR description.

<!-- END -->
