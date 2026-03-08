{{/* See docs/prompts/prime.md */}}
{{ if .WorktreeDirty -}}
> [!WARNING]
> The working tree has uncommitted changes. Ask the user what to do with them before starting new work. Do not begin research or coding until this is resolved.

{{ end -}}
# Beadwork

Beadwork persists plans, progress, and decisions to git so they survive. Compaction erases context.

Issues live on the `beadwork` branch. IDs: `{{ .Prefix }}-XYZ`. Status: open → in_progress → closed / deferred. Priority: P0-P4 (default P2). Epics have children (`--parent`) and deps (`bw dep add <blocker> blocks <blocked>`). `bw ready` = next unblocked work.

## How Should This Land?

Before starting work, ask the user how they want it delivered:

- **Quick fix**: Just make the change in the working tree. No ticket needed.
- **Branch/PR**: Create a ticket first (`bw create "Title" --description "..." -t task`) and work in a worktree. This is the only way to land cleanly.
- **Multi-step**: Create an epic with children and dependencies (see below).

If the user doesn't specify, default to asking. The cost of asking once is lower than the cost of delivering work in the wrong form.

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

1. **Worktree**: Create a worktree with branch `<id>/<short-description>` (e.g. `{{ .Prefix }}-a1b/fix-auth-bug`)
2. **Claim**: `bw start <id>`
3. **Work**: One ticket, one worktree
4. **Land**: Commit with ticket ID → `bw close <id>` → `bw sync`
5. **Clean up**: Remove worktree

What isn't committed, closed, and synced is gone next session.

**Delegation**: Each delegated task needs its own worktree — without isolation, agents corrupt each other's state. Agents that can't request approvals can't land work — plan accordingly. Create a ticket for each delegated task first, then include in the agent prompt:

```
Run `bw start <id>`. Make the change. Commit referencing <id>. Run `bw close <id>`.
```

Verify the work landed after the agent returns.

`bw comment <id> "..."` = breadcrumbs. `bw --help` for everything. `--json` gets you raw data.

## Where You Are

{{ if .Git.IsWorktree }}Worktree{{ else }}Branch{{ end }} `{{ .Git.Branch }}`{{ if .Git.Dirty }} · **uncommitted changes**{{ else }} · clean{{ end }} · last commit: `{{ .Git.LastCommit }}`

## Currently available work:

{{ bw "ready" "--no-context" }}

## Work In Progress

{{ bw "list" "--status" "in_progress" }}
