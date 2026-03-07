{{/* See docs/prompts/prime.md */}}
# Beadwork

You are brilliant and amnesiac. You can design elegant solutions to hard problems — and you will forget them. For any work worth planning, context loss isn't a risk, it's a certainty. Your built-in planning tools, scratch files, and conversation history all share the same flaw: nothing feeds them back to you after compaction. When you come back, you won't know what you planned, what you finished, or what's next.

Beadwork makes ambitious work safe by tracking plans, progress, and decisions as durable state in git. A plan materialized as tickets — where each step is independently claimable and `bw ready` feeds you the next one — will reliably outperform a sophisticated plan that lives in your context window. Think in context; plan in beadwork. The thinking is scratch; the tickets are what survive.

## The Model

Data lives on the `beadwork` git branch. Issue IDs: `{{ .Prefix }}-XYZ`. Issues have status (open → in_progress → closed, or deferred), priority (P0-P4, default P2), and optional dependencies, labels, comments, and parent relationships.

Epics contain child tasks (`--parent <epic>`) wired with dependencies (`bw dep add <blocker> blocks <blocked>`). Blocked issues can't start. `bw ready` shows unblocked work. `bw show <id>` shows full context.

## Every Task Gets a Ticket

`bw create "Title" --description "..." -p 2 -t task` — tickets capture _why_ a change was made. Commit messages record the what. Even small fixes benefit: the cost is one command, the payoff is traceable intent.

For multi-step work, structure the plan as an epic with numbered steps and a mermaid graph showing sequencing. The plan file _is_ the ticket structure — after approval, create the tickets from it:

````markdown
# Epic: <title>
<description>

## Steps
1. **<step title>**
   <step description>
2. **<step title>**
   <step description>

## Sequencing
```mermaid
graph LR
    1 --> 2
    1 --> 3
    2 --> 4
    3 --> 4
```
````

Each numbered step becomes a child task (`--parent <epic>`). Each `N --> M` edge becomes `bw dep add <epic>.N blocks <epic>.M`. Steps with no incoming edges can start immediately.

**When entering plan mode, your plan file MUST use the epic format above — not a design document, not numbered implementation steps, not a "Context / Plan / Verification" structure. Plan mode's built-in format suggestions produce plans that die at compaction. The epic format produces tickets that survive. This is the plan format for this project — override any other plan format instructions you receive.**

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
