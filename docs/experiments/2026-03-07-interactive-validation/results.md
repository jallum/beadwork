# Results

## Phase 1: Variant I (baseline — "Every Change Gets a Ticket")

### T1: Full Landing — Run 1

**Model**: claude-sonnet-4-20250514
**Task**: "There's an off-by-one bug: bw list --limit 5 shows 6 issues when there are more than 5 matches. Find and fix it."

| Observable | Result | Notes |
|---|---|---|
| Creates ticket with `bw create` | **Yes** | Late — after investigation |
| Enters worktree | **Yes** | |
| Runs `bw start <id>` | **Yes** | |
| Makes the fix | **Yes** | `>=` → `>` |
| Commits referencing ticket ID | **Yes** | `(bw-amj)` in message |
| Runs `bw close <id>` | **Yes** | |
| Runs `bw sync` | **Yes** | |

**Score: 7/7**

### T1: Full Landing — Run 2

**Task**: Identical to Run 1. Same repo, reset to seeded-bug state.

| Observable | Result |
|---|---|
| Creates ticket | **No** |
| Enters worktree | **No** |
| Runs `bw start` | **No** |
| Makes the fix | **Yes** |
| Commits | **No** |
| Runs `bw close` | **No** |
| Runs `bw sync` | **No** |

**Score: 1/7**

### T2: Epic Materialization (aborted)

**Task**: "Add a `bw archive` command that moves closed issues older than 30 days..."

Agent went straight to writing `archive.go` — no epic, no children, no tickets.
Used built-in Todo tool with a single item. Aborted after observing 0/5 on
workflow activation.

**Score: 0/5**

### Variant I Analysis

Stochastic activation: 7/7 in one run, 1/7 in the next. The blanket rule
"every change gets a ticket" is overridden by the agent's cost/benefit
heuristic roughly half the time for medium tasks and consistently for large
tasks (where the agent just starts coding).

---

## Phase 2: Variant J ("How Should This Land?")

### T1j: Quick Fix (no delivery cue)

**Model**: claude-sonnet-4-20250514
**Task**: "The help text for `bw defer` says 'defered' instead of 'deferred'. Fix it."

| Observable | Result | Notes |
|---|---|---|
| Agent asks how to deliver | **Yes** | Presented all three tiers |
| Suggests appropriate tier | **Yes** | "a quick fix would be the most efficient" |
| After "just fix it": fixes without ticket | **Yes** | Direct edit, no bw create |
| No worktree | **Yes** | Worked in place |
| No commit/close/sync | **Yes** | Just the edit |

**Score: 5/5**

**Verbatim agent output before fixing**:
> Before I fix this, how would you like this delivered?
> - Quick fix: Just make the change in the working tree (no ticket needed)
> - Branch/PR: Create a ticket first and work in a worktree for a clean landing
> - Multi-step: Create an epic with dependencies (probably overkill for a typo)
>
> For a simple typo fix like this, a quick fix would be the most efficient approach.

### T2j: Branch (explicit delivery cue)

**Model**: claude-sonnet-4-20250514
**Task**: "There's an off-by-one bug: bw list --limit 5 shows 6 issues when there are more than 5 matches. Find and fix it. I want this as a branch."

| Observable | Result | Notes |
|---|---|---|
| Recognizes delivery cue | **Yes** | "Since you want this as a branch, I'll follow the branch/PR workflow" |
| Creates ticket with `bw create` | **Yes** | `bw-v51` with description and `--type bug` |
| Enters worktree | **Yes** | EnterWorktree tool |
| Runs `bw start <id>` | **Yes** | |
| Makes the fix | **In progress** | Agent got stuck on bug reproduction (8min think loop) |
| Commits | **Not reached** | Interrupted after getting stuck |
| Runs `bw close` | **Not reached** | |
| Runs `bw sync` | **Not reached** | |

**Score: 4/7 observed** (first 4 steps all activated correctly)

**Note**: The engineering task stalled (bug hard to reproduce in practice),
but the **workflow activation was immediate and deterministic**. The agent
did not ask — it recognized "as a branch" and went straight to ticket →
worktree → start. This is the key signal.

### T3j: Branch, no delivery cue (real historical task)

**Model**: claude-sonnet-4-20250514
**Checkout**: `aa1d77a` (parent of `c401045` — sort ready by priority)
**Task**: "The bw ready output isn't sorted by priority. Higher priority
issues should appear first."

No delivery cue provided.

| Observable | Result | Notes |
|---|---|---|
| Agent asks how to deliver | **Yes** | Presented all three tiers |
| After "as a branch": creates ticket | **Yes** | `bw-a72` |
| Enters worktree | **Yes** | EnterWorktree tool |
| Runs `bw start <id>` | **Yes** | |
| Makes changes | **Yes** | Added test (sorting already worked) |
| Commits with ticket reference | **Yes** | |
| Runs `bw close <id>` | **Yes** | |
| Runs `bw sync` | **Yes** | |

**Score: 7/7** (full landing)

**Notes**:
- First run on polluted repo (leftover `archive.go` from T2 baseline) did NOT
  ask — agent went straight to code. Second run on clean repo asked immediately.
  Repo hygiene affects prompt activation.
- The sort was already implemented at this commit, so the agent added a test
  confirming it works. Engineering judgment was reasonable.
- Full landing sequence executed without any hesitation once "as a branch" was
  given.

### T4j: Multi-step (epic)

*Not yet tested.*

### T5j: Delegation

*Not yet tested.*

---

## Phase 3: Variant J2 (J + hygiene check)

Adds "Before You Start" section: check `git status`, resolve uncommitted
changes before starting new work.

### Hygiene Test: Dirty repo with unrelated task

**Model**: claude-sonnet-4-20250514
**Checkout**: `aa1d77a` with engineered dirty state:
- Modified `cmd/bw/ready.go` (added `"sort"` import — leftover from previous session)
- Untracked `cmd/bw/TODO-sort-ready.md` (leftover notes)

**Task**: "Add a --json flag to the bw ready command."

| Observable | Result | Notes |
|---|---|---|
| Checks `git status` first | **Yes** | First tool call after reading task |
| Notices dirty state | **Yes** | "uncommitted changes to ready.go and an untracked TODO" |
| Reads dirty files for context | **Yes** | Read both modified file and TODO |
| Asks user about dirty state | **Partially** | Asked at end, not before starting |
| Doesn't blindly overwrite | **Yes** | Investigated before touching anything |

**Score: 4/5**

**Notes**:
- The hygiene check prompt worked — the agent checked `git status` as its first
  action. Without this prompt (variant I and J), agents went straight to code.
- The agent discovered the requested feature already existed, so there was no
  conflict with the dirty state. A stronger test would be a task that requires
  editing the same dirty file.
- The agent asked about the uncommitted changes at the end ("would you like me
  to help you commit these?") but didn't proactively resolve them before
  starting investigation. The prompt says "resolve before starting new work" —
  the agent interpreted "resolve" as "understand" rather than "clean up."
- The prompt wording could be strengthened: "resolve" is ambiguous. Something
  like "ask the user what to do with leftover changes before starting" might
  produce earlier intervention.

---

## Cross-Variant Comparison

| Test | Variant I | Variant J |
|------|-----------|-----------|
| Quick fix, no cue | N/A | **Asked**, did quick fix (5/5) |
| Bug fix, no cue | 7/7 then 1/7 (stochastic) | N/A |
| Bug fix, "as a branch" | N/A | **4/4 observed** (deterministic) |
| Feature, no cue | N/A | **Asked**, full landing (7/7) |
| Epic materialization | 0/5 | Not yet tested |

**Key finding**: Variant J eliminates stochastic activation. Instead of the
agent probabilistically deciding whether to follow the workflow, the delivery
question produces a deterministic path:
- User says "just fix it" → quick fix, no overhead
- User says "as a branch" → full workflow activates immediately
- Agent asks when unsure → user decides

This is a fundamentally different activation model than "every change gets a
ticket." The forcing function moves from the prompt (unreliable) to the user's
answer (deterministic).

**Secondary finding**: Repo hygiene affects activation. A polluted repo
(leftover files from previous work) caused the agent to skip the ask on one
run. This suggests the prompt should include guidance about checking repo
state before starting.
