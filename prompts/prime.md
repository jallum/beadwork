{{/* See docs/prompts/prime.md */}}
# Beadwork

Beadwork tracks work as durable state in git — plans, progress, and decisions that survive compaction, session boundaries, and context loss. Your built-in planning tools handle small tasks; beadwork makes ambitious work safe by checkpointing progress so you can pick up after losing context.

Plans in your context window die at compaction. Make them durable: a comment on the issue preserves a simple plan; an epic with children and dependencies preserves a complex one. `bw ready` feeds you the next step — you don't need to remember it all.

## The Model

Data lives on the `beadwork` git branch. Issue IDs: `{{ .Prefix }}-XYZ`. Issues have status (open → in_progress → closed, or deferred), priority (P0-P4, default P2), and optional dependencies, labels, comments, and parent relationships.

Epics contain child tasks (`--parent <epic>`) wired with dependencies (`bw dep add <blocker> blocks <blocked>`). Blocked issues can't start. `bw ready` shows unblocked work. `bw show <id>` shows full context.

## Every Task Gets a Ticket

`bw create "Title" --description "..." -p 2 -t task` — tickets capture _why_ a change was made. Commit messages record the what. Even small fixes benefit: the cost is one command, the payoff is traceable intent. Larger efforts: epic with children so `bw ready` feeds you the next step.

## Starting Work

Beadwork's issue state is concurrency-safe; the repo's working tree is not. The main working tree belongs to the user — working directly in it blocks them from other tasks, other branches, other agents.

**The workflow:**
1. **Enter a worktree** for the ticket — use the word "worktree" when telling the user what you're doing (e.g., "I'll work in a worktree for {{ .Prefix }}-xyz"). This activates the EnterWorktree tool, giving you an isolated branch at `.claude/worktrees/<name>`.
2. **Claim the ticket**: `bw start <id>` — sets status, assigns you, shows context and landing instructions.
3. **Do the work.** One ticket per worktree. Related fixes are new tickets, not scope expansion.
4. **Land it** per `bw start`'s instructions. Work that isn't committed, closed, and synced is invisible to the next session. Remove the worktree when done.

If a `bw` command fails, read the error — beadwork errors are descriptive and actionable.

## Delegation

Sub-agents don't inherit your context — they won't use worktrees or leave breadcrumbs unless you tell them to. Include the workflow in the handoff:

> **Setup:** Enter a worktree for the ticket, then run `bw start <id> --assignee <agent-id>`. This claims the ticket and delivers your full briefing — what to build, how to land it, everything. Follow its instructions.
>
> **When done:** `bw comment <id> "summary of what you did"` before closing.

Close the ticket only after verifying the work landed.

## Durable Notes

`bw comment <id> "..."` — breadcrumbs for your future self after compaction, and messages to anyone else working in the project.

## Commands

`bw --help` lists everything. `--help` on any subcommand. Read commands accept `--json`. Common: `bw list --grep "auth"`, filter by `--status`/`--label`/`--assignee`. `bw defer <id> 2026-03-01`. `bw delete <id>` (previews first; `--force` to confirm).

## Currently available work:

{{ bw "ready" }}
## Work In Progress

{{ bw "list" "--status" "in_progress" }}
