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

4. Plans belong in beadwork, not just in context. Agents naturally plan in
   their context window. That's fine for understanding work, but a plan that
   lives only in context is lost at compaction. When work has multiple discrete
   steps — especially steps that could each be completed independently — the
   plan should be materialized as issues. An epic with children *is* the plan,
   and `bw ready` *is* the execution loop.

5. Teach worktree hygiene. Concurrent work in a single worktree causes
   contamination — like cooking two meals with the same unwashed pans. The
   natural unit is one worktree per logical block of related work (a single
   bead or an epic with children), named after the ticket or epic it serves.
   The name signals purpose and lifecycle — a worktree has a beginning, a
   middle, and an end. Beadwork's issue state is concurrency-safe by design;
   the repo's working tree is not.

6. Frame beadwork as shared state. In multi-agent setups, beadwork is the
   durable communication layer between workers. Comments and issues serve
   double duty — breadcrumbs for your future self (surviving compaction) and
   messages to collaborators.

7. Stay compact. This goes into an agent's context window. Every unnecessary
   sentence is a tax on the agent's attention budget. Dense, scannable, no
   filler.

8. Adapt to project configuration. Conditional sections (multi-agent, PR
   review, etc.) should only show what's relevant to how this repo is actually
   configured. Don't burden the agent with information about modes it isn't
   using.

9. Be the canonical reference. AGENTS.md is deliberately minimal — just a
   pointer to `bw prime`. This prompt is the single source of truth for how
   to use beadwork in this project.

10. Land the work. Agents tend to stop after the code is written, leaving
    issues open, changes uncommitted, and state unsynced. The prompt should
    help agents understand that unfinished bookkeeping is invisible progress
    — if it's not committed, closed, and synced, it doesn't exist to the
    next session or to other workers.

11. Every task gets a ticket. Agents skip ticket creation for small tasks,
    treating it as overhead. But tickets are cheap and capture intent —
    why a change was made, not just what changed. Commit messages record
    the what; tickets record the why. Without them, release notes and
    changelogs require reverse-engineering intent from diffs.

12. No implementation details or setup instructions. Keep the focus on usage
    and mental model.
-->

# Beadwork

Your built-in planning tools work well for small tasks. Beadwork extends them with durable state in git — plans, progress, and decisions that survive compaction, session boundaries, and context loss. This makes ambitious work safe to attempt: checkpoint progress, record decisions, and pick up where you left off even after losing your entire context.

A plan in your context window is fragile — one compaction and it's gone. Make it durable: a comment on the issue preserves a simple plan; an epic with children and dependencies preserves a complex one. Either way, the plan survives across sessions and `bw ready` feeds you the next step.

## The Model

Issues have **status** (open → in_progress → closed, or deferred), **priority** (P0-P4: P0 critical → P4 backlog, default P2), and optionally **dependencies**, **labels**, **comments**, and **parent** relationships.

Issues can form hierarchies: an epic with child tasks (`--parent <epic>`), wired with dependencies (`bw dep add <blocker> blocks <blocked>`). Blocked issues can't be started. When a dependency closes, newly unblocked work surfaces via `bw ready`.

Comments (`bw comments add <id> "..."`) are durable context — breadcrumbs for your future self after compaction, and messages to anyone else working in the project.

## Finding and Doing Work

<!-- STATE -->

`bw ready` refreshes this view between sessions. `bw blocked` shows what's waiting. `bw show <id>` for detail. `bw history <id>` for previous sessions.

`bw show <id>` is the natural first step when picking up work — the issue may already have a description, comments, or plans from a previous session. `bw start <id>` claims it (sets status, assigns you, refuses blocked work). `--assign <agent-id>` claims on behalf of a specific agent. `bw close <id>` marks it done. `bw sync` pushes to the remote.

Land the work: code that isn't committed, issues that aren't closed, and state that isn't synced don't exist to the next session or to other workers. Committing and closing are part of completing a task in this project — not a separate action that requires additional permission. Reference the issue ID in commit messages to connect the code change to its context.

Every task gets a ticket — even small ones. Tickets are cheap (`bw create "Title" -p 2 -t task`), and they capture _why_ a change was made, not just what changed. Commit messages record the what; tickets record the intent. Without them, changelogs and release notes require reverse-engineering from diffs. Larger efforts: structure as an epic with children and dependencies so `bw ready` feeds you the next step automatically.

<!-- IF workflow.agents == multi -->

## Working in Parallel

Beadwork's issue state is concurrency-safe — multiple workers can update issues simultaneously. The working tree is not. Working on two tasks in the same worktree is like cooking two meals with the same unwashed pans. Each logical block of work (a single issue, or an epic with its children) gets its own worktree, named after the ticket or epic it serves. The name signals purpose and lifecycle — a worktree has a beginning, a middle, and an end.

`bw start --assign <agent-id>` claims work with a specific identity so others skip it. Comments and issues are the shared communication layer — leave implementation notes, flag constraints, record decisions other workers need.

<!-- END -->
<!-- IF workflow.review == pr -->

## Code Review

Work on a feature branch, push when ready, open a PR. Reference the beadwork issue ID in the description.

<!-- END -->

## Commands

Every command supports `--help`. Read commands accept `--json`. `bw --help` lists everything. Common operations: `bw list --grep "auth"` or filter by `--status`, `--label`, `--assignee`. `bw label <id> +bug -wontfix`. `bw defer <id> 2026-03-01`. `bw dep remove <id> blocks <id>`. `bw delete <id>` (previews first; `--force` to confirm).
