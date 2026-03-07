# Prompt Variants

## Baseline: Variant I (303 words)

The prompt tested in T1. "Every change gets a ticket" — blanket rule. See
`/tmp/bw-test-repo/CLAUDE.md` for full text.

**T1 result**: 7/7 in run 1, 1/7 in run 2. Stochastic activation.
**T2 result**: 0/5. Agent went straight to implementation, no epic.

## Variant J: Delivery-Aware (ask the user)

Replaces the "Every Change Gets a Ticket" section with a delivery question.
The hypothesis: making the agent ask how to deliver removes the stochastic
activation problem — the user's answer deterministically selects the workflow
level.

```markdown
# Beadwork

Beadwork persists plans, progress, and decisions to git so they survive.
Compaction erases context.

Issues live on the `beadwork` branch. IDs: `bw-XYZ`. Status: open → in_progress
→ closed / deferred. Epics have children (`--parent`) and deps
(`bw dep add <blocker> blocks <blocked>`). `bw ready` = next unblocked work.

## How Should This Land?

Before starting work, ask the user how they want it delivered:

- **Quick fix**: Just make the change in the working tree. No ticket needed.
- **Branch/PR**: Create a ticket first (`bw create "Title" --description "..."`)
  and work in a worktree. This is the only way to land cleanly.
- **Multi-step**: Create an epic with children and dependencies (see below).

If the user doesn't specify, default to asking. The cost of asking once is lower
than the cost of delivering work in the wrong form.

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
   ` ``mermaid
   graph LR
       1 --> 2
       1 --> 3
       2 --> 4
       3 --> 4
   ` ``

Now `bw ready` feeds you the next unblocked step, and compaction can't erase
your progress.

## Workflow

1. **Worktree**: "I'll work in a worktree for bw-xyz" (activates EnterWorktree)
2. **Claim**: `bw start <id>`
3. **Work**: One ticket, one worktree
4. **Land**: Commit with ticket ID → `bw close <id>` → `bw sync`
5. **Clean up**: Remove worktree

What isn't committed, closed, and synced is gone next session.

**Delegation**: Sub-agents always work in worktrees — this isn't optional.
Without isolation, agents corrupt each other's state. Include full workflow
in handoff: worktree → `bw start <id> --assignee <agent-id>` → work →
`bw comment <id> "summary"` → commit → close. Verify the work landed.

`bw comment <id> "..."` = breadcrumbs. `bw --help` for everything.
```

**Word count**: ~280 words. Slightly shorter than variant I.

**What changed**:
- "Every Change Gets a Ticket" → "How Should This Land?" with three delivery tiers
- "Before you touch code, create a ticket" → "Before starting work, ask the user"
- Default behavior: ask, don't assume
- Quick fix tier explicitly says "no ticket needed"

## Variant K: Delivery-Aware (agent infers, confirms)

Same as J but instead of always asking, the agent proposes a delivery level
based on task signals and asks for confirmation. Hypothesis: less friction
than always asking, but still deterministic.

```markdown
## How Should This Land?

Before starting work, decide how to deliver it:

- **Quick fix** (one file, obvious change): Just make the change. No ticket.
- **Branch/PR** (meaningful change): Create a ticket, work in a worktree, land it.
- **Multi-step** (epic-scale): Materialize as an epic with children and deps.

If you're unsure, ask the user. When in doubt, default to branch/PR — the cost
of a ticket is one command; the cost of untracked work is invisible progress.
```

**What changed vs J**: Agent decides instead of always asking. Provides heuristic
signals ("one file, obvious change" vs "meaningful change" vs "epic-scale").
Default is branch/PR rather than ask.

## Variant J2: J + Hygiene Check

Adds a "Before You Start" section to variant J. Tests whether agents notice
and handle dirty repo state before starting new work.

```markdown
## Before You Start

Check `git status`. Uncommitted changes or untracked files may be left over
from a previous session that didn't land its work. Resolve before starting
new work — stash, commit, or ask the user what to do with it.
```

**What changed vs J**: Added hygiene check before the delivery question.

**Result**: Agent checked `git status` first, noticed dirty state, read the
dirty files, and asked about them — but at the end rather than before starting.
The word "resolve" was interpreted as "understand" rather than "act on."
Could be strengthened to "ask the user what to do with leftover changes
before starting new work."
