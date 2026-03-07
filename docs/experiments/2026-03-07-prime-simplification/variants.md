# Prompt Variants

## A: Baseline (699 words)

The current prime prompt, unchanged. Literary opening ("brilliant and amnesiac"),
full epic template with 4-edge mermaid example, full plan-mode override with all
5 load-bearing elements, separate sections for Starting Work, Delegation, Durable
Notes, and Commands.

## B: Reordered (699 words, 0% reduction)

Baseline with ready/WIP state moved to the top, before instructional content.
Tests T1 (order matters).

## C: Direct Opening (682 words, 2% reduction)

Replaced "brilliant and amnesiac" paragraph with direct statement: "Context loss
is certain. Every plan, decision, and progress marker in your context window will
be lost at compaction." Otherwise unchanged. Tests T3 (opening replaceable).

## D: Consistent Tone (690 words, 1% reduction)

Authoritative register throughout. Bold lead sentences on every section. Lighter
plan override (dropped "MUST", restructured sentence). Tests T2+T6.

## E: Compressed (369 words, 47% reduction)

Merged sections (Delegation into Working, Durable Notes inline). Tighter template.
Compressed workflow into prose paragraph instead of numbered list. Tests T4+T5.

## F: Hybrid (405 words, 42% reduction)

E's body compression + A's numbered workflow list (5 steps including worktree
cleanup). Full plan-mode override preserved.

## G: Aggressive (283 words, 60% reduction)

Tighter epic template (1-line steps, 2-edge graph), terser prose throughout.
Numbered workflow preserved. Full plan-mode override.

## H: Minimal (224 words, 68% reduction)

Absolute minimum. Telegraphic style. Compressed plan override (kept "MUST" but
shorter). No separate Model section — data model folded into opening.

## I: Additive (303 words, 57% reduction)

**Fundamentally different approach.** Removes the plan-mode override entirely.
Instead of forcing epic format, adds a materialization step: "Plan however you
want. Before you start executing, materialize it into beadwork." Numbered steps
for creating epic, children, and deps. "Before you touch code, create a ticket"
as activation trigger.

````
# Beadwork

Beadwork persists plans, progress, and decisions to git so they survive.
Compaction erases context.

Issues live on the `beadwork` branch. IDs: `bw-XYZ`. Status: open → in_progress
→ closed / deferred. Epics have children (`--parent`) and deps
(`bw dep add <blocker> blocks <blocked>`). `bw ready` = next unblocked work.

## Every Change Gets a Ticket

Before you touch code, create a ticket:
`bw create "Title" --description "..." -t task`. Tickets capture _why_ a change
was made — intent that commits don't preserve. Even a one-line fix gets a ticket;
the cost is one command.

## Plans Are Scratch — Tickets Survive

Plan however you want. Your plan is useful for thinking, but it lives in context
and dies at compaction. Before you start executing a plan, materialize it into
beadwork:

1. Create an epic: `bw create "Epic title" -t epic --description "..."`
2. Create a child task for each step:
   `bw create "Step title" --parent <epic> --description "..."`
3. Wire dependencies: `bw dep add <blocker> blocks <blocked>`
4. Include a mermaid sequencing graph in the epic description so the dependency
   structure is visible:
   ```mermaid
   graph LR
       1 --> 2
       1 --> 3
       2 --> 4
       3 --> 4
````

Now `bw ready` feeds you the next unblocked step, and compaction can't erase
your progress.

## Workflow

1. **Worktree**: "I'll work in a worktree for bw-xyz" (activates EnterWorktree)
2. **Claim**: `bw start <id>`
3. **Work**: One ticket, one worktree
4. **Land**: Commit with ticket ID → `bw close <id>` → `bw sync`
5. **Clean up**: Remove worktree

What isn't committed, closed, and synced is gone next session.

**Delegation**: Sub-agents lack your context. Include workflow in handoff:
worktree → `bw start <id> --assignee <agent-id>` → work →
`bw comment <id> "summary"` → close. Verify.

`bw comment <id> "..."` = breadcrumbs. `bw --help` for everything.

```

```
