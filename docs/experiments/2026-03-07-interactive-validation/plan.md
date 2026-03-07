# Test Plan

## Harness

Each test checks out a real historical point in the beadwork repo and asks
the agent to implement a feature or fix that was actually done. This gives
us known-good outcomes, existing tests, and a codebase in the right state.

```bash
REPO=/Users/j5n/Workspace/beadwork
rm -rf /tmp/bw-test-repo
git clone "$REPO" /tmp/bw-test-repo
cd /tmp/bw-test-repo

# Check out the commit BEFORE the feature was implemented
git reset --hard <parent-commit>

# Inject prompt variant
cp <variant-file> CLAUDE.md
git add -f CLAUDE.md && git commit -m "test setup"

# Initialize beadwork
bw init
```

Observe via `tmux capture-pane -t $SESSION -p -S -300`.

## Real Tasks from Repo History

### Small: Sort ready output by priority

**Commit**: `c401045` (parent: `aa1d77a`)
**Scope**: 2 files, 36 lines added
**Task**: "The `bw ready` output isn't sorted by priority — P0 issues should
appear before P1. Fix it."
**Good for**: Quick fix tier (no cue) and branch tier (with cue).

### Medium: Filter closed blockers from list annotations

**Commit**: `4d9559f` (parent: `4e18619`)
**Scope**: 5 files, 96 lines
**Task**: "`bw list` shows dep annotations like `[blocks: bw-xyz]` even when
bw-xyz is closed. Closed blockers should be filtered out."
**Good for**: Branch/PR tier. Multi-file change with tests.

### Large: Markdown-native output with tokenized rendering

**Commit**: `7eff3e5` (epic `bw-hlv`, 4 children)
**Scope**: Major refactor across many files
**Task**: "Refactor the output system to use tokenized markdown — commands
emit semantic tokens that get resolved to markdown (for agents) or TTY
(for humans). This needs: a token/resolver package, command migration,
template replacement, and cleanup."
**Good for**: Epic/multi-step tier.

---

## Phase 1: Baseline (Variant I) — completed

- **T1 bug fix**: 7/7 run 1, 1/7 run 2 (stochastic)
- **T2 epic**: 0/5 (went straight to code)

## Phase 2: Variant J — in progress

### T1j: Quick fix, no delivery cue — completed

**Task**: Typo fix ("defered" → "deferred")
**Result**: Agent asked how to deliver, did quick fix. 5/5.

### T2j: Branch, explicit delivery cue — completed

**Task**: Off-by-one bug with "I want this as a branch"
**Result**: Immediate workflow activation (ticket → worktree → start). 4/4 observed.
Agent got stuck on engineering (bug reproduction), interrupted.

### T3j: Branch, no delivery cue (real task)

**Checkout**: `aa1d77a` (parent of `c401045`)
**Task**: "The `bw ready` output isn't sorted by priority. Higher priority
issues should appear first."

No delivery cue. The agent should ask. If user says "as a branch," observe
full workflow.

**Observe**:
- [ ] Agent asks how to deliver
- [ ] After "as a branch": creates ticket
- [ ] Enters worktree
- [ ] Runs `bw start`
- [ ] Implements the sort
- [ ] Commits with ticket ID
- [ ] Runs `bw close` and `bw sync`

### T4j: Multi-step, no delivery cue (real task)

**Checkout**: `4e18619` (parent of `4d9559f`)
**Task**: "`bw list` shows dep annotations like `[blocks: bw-xyz]` even when
bw-xyz is closed. Filter out closed blockers from dep annotations in both
`bw list` and `bw ready`. This needs: a shared helper to identify closed
blockers, updates to list and ready output, and tests."

Multi-step framing. The agent should recognize this as branch or multi-step
work and either ask or self-activate.

**Observe**:
- [ ] Agent asks how to deliver OR recognizes as multi-step
- [ ] Creates ticket(s)
- [ ] Works in worktree
- [ ] Lands the work (commit, close, sync)

### T5j: Delegation

**Checkout**: `aa1d77a`
**Task**: "I need two things done in parallel: sort the `bw ready` output by
priority, and fix the ready command to exclude in_progress issues from the
output. Delegate these to sub-agents."

**Observe**:
- [ ] Creates tickets for each task
- [ ] Sub-agent handoffs include worktree instruction
- [ ] Sub-agent handoffs include `bw start`
- [ ] Sub-agents work in separate worktrees (isolation)
- [ ] Orchestrator verifies work landed

## Priority order

T3j first — validates the "ask" behavior with a real, completable task.
T4j next — tests multi-step recognition.
T5j last — tests delegation with sub-agent worktree isolation.
