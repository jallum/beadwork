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

8. Adapt to project configuration. Per-task conditionals (PR review, etc.)
   now live in start.md and render at point of action. Prime shows the full
   mental model to all agents regardless of configuration.

9. Be the canonical reference. AGENTS.md is deliberately minimal — just a
   pointer to `bw prime`. This prompt is the single source of truth for how
   to use beadwork in this project.

10. Land the work. Prime establishes the principle (unfinished bookkeeping
    is invisible progress); `bw start` delivers the concrete steps via
    start.md. Prime should reinforce that landing matters without
    duplicating the procedure.

11. Every task gets a ticket. Agents skip ticket creation for small tasks,
    treating it as overhead. But tickets are cheap and capture intent —
    why a change was made, not just what changed. Commit messages record
    the what; tickets record the why. Without them, release notes and
    changelogs require reverse-engineering intent from diffs.

12. No implementation details or setup instructions. Keep the focus on usage
    and mental model.

13. Teach delegation. When orchestrating sub-agents, the orchestrator has
    beadwork context but the workers don't. The prompt should make clear
    that delegated tasks must include the workflow steps — claim, do, land
    — or the workers will skip them. The orchestrator is responsible for
    including the workflow and verifying the work landed.
-->

# Beadwork

Beadwork is a state-management tool designed specifically to help you get things done more efficiently.

Your built-in planning tools work well for small tasks. Beadwork extends them with durable state in git — plans, progress, and decisions that survive compaction, session boundaries, and context loss. This makes ambitious work safe to attempt: checkpoint progress, record decisions, and pick up where you left off even after losing your entire context.

A plan in your context window is fragile — one compaction and it's gone. Make it durable: a comment on the issue preserves a simple plan; an epic with children and dependencies preserves a complex one. Either way, the plan survives across sessions and `bw ready` feeds you the next step.

## The Model

All data lives on the `beadwork` git branch — deleting it permanently destroys everything.

Issues have **status** (open → in_progress → closed, or deferred), **priority** (P0-P4: P0 critical → P4 backlog, default P2), and optionally **dependencies**, **labels**, **comments**, and **parent** relationships. Issues can form hierarchies: an epic with child tasks (`--parent <epic>`), wired with dependencies (`bw dep add <blocker> blocks <blocked>`). Blocked issues can't be started; when a dependency closes (`bw close <id>`), newly unblocked work surfaces. To find more work, use `bw ready`. `bw show <id>` will give you the description, assignee, comments, blockers and the tickets that _that_ ticket will unblock, all in one go.

## Tickets are Cheap Insurance

Every task gets a ticket (`bw create "Title" --description "..." -p 2 -t task`) to capture _why_ a change was made — commit messages record the what, tickets record the intent. Larger efforts: structure as an epic with children and dependencies so `bw ready` feeds you the next step -- you don't need to remember it all.

## Starting the Work

When working solo as the main agent, `bw start <id>` claims it — sets status, assigns the current user, shows the full issue context, and tells you how to land it when done.

When delegating the work to another agent, remember that they don't inherit your context. Each worker will need to run `bw start <id> --assignee <delegate-agent-id>` itself -- this will explain the issue context and how to land the work, while signalling to other agents _who_ is doing _what_. It's important to close the ticket after you've verified that the agent has completed the work.

Work that isn't committed, closed, and synced doesn't exist to the next session or to other workers. Landing instructions are delivered by `bw start` — they aren't embedded in tickets or stored in context. A plan that skips these steps leaves the worker without a path to land.

## Leaving Notes

Comments (`bw comment <id> "..."`) are durable context — breadcrumbs for your future self after compaction, and messages to anyone else working in the project. This is especially useful for leaving implementation notes when deviations or adjustments are necessary. 

## Worktrees

Beadwork's issue state is concurrency-safe; the main working tree is not.

Each logical block of work should be done within its own git worktree, named after the ticket it serves (`<ticket-id>`). This includes analysis — the main worktree may be on the wrong branch or dirty with someone else's changes, so plans formed there can start from false assumptions. Skipping this seems faster — especially when you spot a related fix along the way — but mixing two or more tasks in one worktree creates messes that are hard to untangle. That related fix is a new ticket, not an expansion of the current one. Worse, a dirty main worktree blocks the user from starting other tasks, switching branches, or working on anything else. Keeping our work entirely inside a purpose-driven worktree makes us better neighbors. When we're all done with a worktree we've created, we should remove it and leave things as we found them -- the local branch preserves all of the work.

## Commands

Every command supports `--help`. Read commands accept `--json`. `bw --help` lists everything. Common operations: `bw list --grep "auth"` or filter by `--status`, `--label`, `--assignee`. `bw label <id> +bug -wontfix`. `bw defer <id> 2026-03-01`. `bw dep remove <id> blocks <id>`. `bw delete <id>` (previews first; `--force` to confirm).

## Currently available work:

```text
<!-- bw ready -->
```

## Work In Progress

``` text
<!-- bw list --status in_progress -->
```