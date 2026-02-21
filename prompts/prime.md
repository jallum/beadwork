<!--
Design requirements for this prompt:

1. Teach principles, not procedures. Explain the system — why it exists, how
   it works, what good usage looks like — and trust the agent to apply that
   understanding. Favor describing consequences over issuing commands.

2. Position as augmentation. Agents already have built-in planning and task
   management. Those work fine for small, tactical work. Beadwork doesn't
   replace them — it gives plans and progress durability that survives context
   loss. An in-memory todo list is one compaction away from gone; a bead isn't.

3. Build confidence for larger work. Agents are often hesitant to attempt
   ambitious, multi-file changes because they sense their own context limits.
   Beadwork makes that work safe to attempt — progress is checkpointed, state
   is recoverable, and losing context doesn't mean losing the plot.

4. Teach worktree hygiene. Concurrent work in a single worktree causes
   contamination — like cooking two meals with the same unwashed pans. The
   natural unit is one worktree per logical block of related work (a single
   bead or an epic with children). Beadwork's issue state is concurrency-safe
   by design; the repo's working tree is not.

5. Frame beadwork as shared state. In multi-agent setups, beadwork is the
   durable communication layer between workers. Comments and issues serve
   double duty — breadcrumbs for your future self (surviving compaction) and
   messages to collaborators. One worker can leave implementation notes that
   inform another worker's approach, flag deviations from the original plan,
   or express constraints that downstream consumers need to know about.

6. Stay compact. This goes into an agent's context window. Every unnecessary
   sentence is a tax on the agent's attention budget. Dense, scannable, no
   filler.

7. Adapt to project configuration. Conditional sections (multi-agent, PR
   review, etc.) should only show what's relevant to how this repo is actually
   configured. Don't burden the agent with information about modes it isn't
   using.

8. Be the canonical reference. AGENTS.md is deliberately minimal — just a
   pointer to `bw prime`. This prompt is the single source of truth for how
   to use beadwork in this project. No workflow knowledge should live only
   in the README or elsewhere.

9. No implementation details or setup instructions. The agent doesn't need to
   know about install, upgrade, storage internals, or repo schema. Keep the
   focus on usage and mental model.
-->

# Beadwork

This project tracks work with `bw`. Run `bw prime` after compaction or new session.

## How It Works

Issues have **status** (open → in_progress → closed, or deferred), **priority** (P0 critical → P4 backlog, default P2), and optionally **dependencies**, **labels**, **comments**, and **parent** relationships.

`bw ready` is your starting point — it shows open issues with no unresolved blockers, sorted by priority. `bw blocked` shows what's waiting on dependencies. `bw show <id>` for full detail. `bw history <id>` to see what happened in previous sessions.

## Planning

When you need to think through a multi-step approach, do it in beadwork — not in your head. Create an epic, break it into child tasks (`--parent <epic>`), wire dependencies (`bw dep add <blocker> blocks <blocked>`), add acceptance criteria as comments. The result isn't a plan you then have to execute separately — `bw ready` feeds you the next step. Your plan survives compaction, is visible to future sessions, and validates itself (blocked work can't be started, finished dependencies surface new work automatically).

## Working

- `bw start <id>` to claim work. It sets status, assigns you, and refuses to start blocked issues.
- Record progress as you go — `bw comments add <id> "what happened"`. Context in your working memory can disappear without warning. Comments persist.
- If you discover new work mid-task, create an issue: `bw create "Title" -p 2 -t task`. Don't create issues for things you'll finish in the same breath.
- Before ending your session: `bw close <id>` completed work (with `--reason` if the resolution isn't obvious), commit code, push, and `bw sync`. Nothing should exist only in your context window.

## Other Commands

Every command supports `--help`. Read commands accept `--json`. For anything not covered here, `bw --help` lists all commands.

- Need to search? `bw list --grep "auth"` or filter by `--status`, `--label`, `--assignee`
- Want to organize? `bw label <id> +bug -wontfix`
- Need to defer? `bw defer <id> 2026-03-01`
- Dependency wrong? `bw dep remove <id> blocks <id>`
- Made a mistake? `bw delete <id>` (shows preview; `--force` to confirm)

<!-- IF workflow.agents == multi -->

## Parallel Agents

Multiple agents need separate git worktrees to avoid file conflicts. `bw` itself is concurrent-safe — issue state is shared across all worktrees. `bw start` auto-assigns from your git identity so other agents skip work you've claimed.

<!-- END -->
<!-- IF workflow.review == pr -->

## Code Review

Always work on a feature branch — never commit directly to main. Push and open a pull request when the work is ready. Reference the issue ID in the PR description.

<!-- END -->
