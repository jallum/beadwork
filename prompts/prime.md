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

Your built-in task tracking and planning tools work well for small, focused tasks. Beadwork extends them — it gives your plans, progress, and decisions a durable home in git that survives compaction, session boundaries, and context loss.

This means work that might otherwise feel risky — large refactors, multi-step features, changes that span many files — becomes safe to attempt. You can checkpoint progress, record decisions, and pick up exactly where you left off, even if you lose everything in your working memory.

## The Model

Issues have **status** (open → in_progress → closed, or deferred), **priority** (P0-P4: P0 critical → P4 backlog, default P2), and optionally **dependencies**, **labels**, **comments**, and **parent** relationships.

Issues can form hierarchies: an epic with child tasks (`--parent <epic>`), wired together with dependencies (`bw dep add <blocker> blocks <blocked>`). Blocked issues can't be started. When a dependency closes, newly unblocked work surfaces automatically via `bw ready`.

`bw show <id>` is more than a detail view — its BLOCKED BY section walks the full dependency tree down to the leaves, then surfaces only the issues that are actionable _right now_. If A depends on B which depends on C which depends on D, and D is the only open leaf, BLOCKED BY shows D — not the whole chain. As issues close, the next layer surfaces. It's like `bw ready` scoped to a specific issue.

Comments (`bw comments add <id> "..."`) serve double duty. They're breadcrumbs for your future self — context that survives compaction — and they're messages to anyone else working in the same project. An implementation note you leave on an issue can inform a collaborator's approach, flag a deviation from the original plan, or express a constraint that downstream work needs to account for.

## Finding and Doing Work

<!-- STATE -->

`bw ready` refreshes this view between sessions — open issues with no unresolved blockers, sorted by priority. `bw blocked` shows what's waiting on dependencies. `bw show <id>` for full detail. `bw history <id>` for what happened in previous sessions.

`bw start <id>` claims an issue — it sets status, assigns you, and won't let you start something that's still blocked. Use `--assign <agent-id>` to claim work on behalf of a specific agent. `bw close <id>` marks it done. `bw sync` pushes state to the remote so nothing exists only in your local copy.

When you discover new work mid-task, `bw create "Title" -p 2 -t task` captures it. For larger efforts, structuring work as an epic with children and dependencies means `bw ready` becomes your plan — it feeds you the next step automatically, and that plan survives across sessions.

<!-- IF workflow.agents == multi -->

## Working in Parallel

Beadwork's issue state is concurrency-safe — multiple workers can read and update issues simultaneously without conflict. The repository's working tree is not. Working on two tasks in the same worktree is like preparing two meals with the same pans without washing them — things get crossed. Each logical block of work (a single issue, or an epic with its children) should have its own git worktree.

`bw start --assign <agent-id>` lets a worker claim issues with its own identity, so other workers can see what's taken and skip it. Comments and issues are the shared communication layer between workers — use them to leave implementation notes, flag constraints, or record decisions that other workers need to know about.

<!-- END -->
<!-- IF workflow.review == pr -->

## Code Review

This project uses pull requests for review. Work on a feature branch, push when ready, and open a PR. Referencing the beadwork issue ID in the PR description connects the code change to its context.

<!-- END -->

## Commands

Every command supports `--help`. Read commands accept `--json`. `bw --help` lists everything, but common operations: `bw list --grep "auth"` or filter by `--status`, `--label`, `--assignee`. `bw label <id> +bug -wontfix`. `bw defer <id> 2026-03-01`. `bw dep remove <id> blocks <id>`. `bw delete <id>` (previews first; `--force` to confirm).
