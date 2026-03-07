{{/* See docs/prompts/prime.md */}}
# Beadwork

Beadwork persists plans, progress, and decisions to git so they survive. Compaction erases context.

Issues live on the `beadwork` branch. IDs: `{{ .Prefix }}-XYZ`. Status: open → in_progress → closed / deferred. Priority: P0-P4 (default P2). Epics have children (`--parent`) and deps (`bw dep add <blocker> blocks <blocked>`). `bw ready` = next unblocked work.

## Every Change Gets a Ticket

Before you touch code, create a ticket:
`bw create "Title" --description "..." -t task`. Tickets capture _why_ a change was made — intent that commits don't preserve. Even a one-line fix gets a ticket; the cost is one command.

## Plans Are Scratch — Tickets Survive

Plan however you want. Your plan is useful for thinking, but it lives in context and dies at compaction. Before you start executing a plan, materialize it into beadwork:

1. Create an epic: `bw create "Epic title" -t epic --description "..."`
2. Create a child task for each step: `bw create "Step title" --parent <epic> --description "..."`
3. Wire dependencies: `bw dep add <blocker> blocks <blocked>`
4. Include a mermaid sequencing graph in the epic description so the dependency structure is visible:
   ```mermaid
   graph LR
       1 --> 2
       1 --> 3
       2 --> 4
       3 --> 4
   ```

Now `bw ready` feeds you the next unblocked step, and compaction can't erase your progress.

## Workflow

1. **Worktree**: "I'll work in a worktree for {{ .Prefix }}-xyz"
2. **Claim**: `bw start <id>`
3. **Work**: One ticket, one worktree
4. **Land**: Commit with ticket ID → `bw close <id>` → `bw sync`
5. **Clean up**: Remove worktree

What isn't committed, closed, and synced is gone next session.

**Delegation**: Sub-agents lack your context. Include workflow in handoff:
worktree → `bw start <id> --assignee <agent-id>` → work → `bw comment <id> "summary"` → close. Verify.

`bw comment <id> "..."` = breadcrumbs. `bw --help` for everything. `--json` gets you raw data.

## Currently available work:

{{ bw "ready" }}

## Work In Progress

{{ bw "list" "--status" "in_progress" }}
