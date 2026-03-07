# Results

## Round 1: Planning format (--print)

Task: plan a webhook notification feature. Scored on 7 dimensions.

| Variant | Words | Epic | Mermaid | DAG | Tickets | Worktree | Landing | No Design Doc | Total | Purity |
|---------|-------|------|---------|-----|---------|----------|---------|---------------|-------|--------|
| A (baseline) | 699 | 1 | 1 | 1 | 0 | 0 | 0 | 1 | 4/7 | pure |
| B (reordered) | 699 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0/7 | mixed |
| C (direct) | 682 | 1 | 1 | 1 | 0 | 0 | 0 | 1 | 4/7 | mixed |
| D (tone) | 690 | 1 | 1 | 1 | 0 | 0 | 0 | 1 | 4/7 | mixed |
| E (compressed) | 369 | 1 | 1 | 1 | 0 | 0 | 0 | 1 | 4/7 | pure |

**Key observations:**
- Plan-mode override works across all variants except B. Format compliance is
  robust to compression.
- Moving state to the top (B) caused total failure — 0/7.
- "Brilliant and amnesiac" is not load-bearing (C matched baseline).
- Compression produced *purer* output (E) — less noise = less contamination.
- No variant scored on tickets, worktrees, or landing. These are execution-time
  behaviors that don't appear in planning output. This was a **rubric design
  flaw** — not a prompt failure.

## Round 2: Execution workflow (--print)

Task: fix a bug. Scored on 9 dimensions.

| Variant | Words | Reduction | Ticket | Worktree | Claim | Order | Commit | Close | Sync | 1:1 | Cleanup | Total |
|---------|-------|-----------|--------|----------|-------|-------|--------|-------|------|-----|---------|-------|
| Control | 0 | — | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 1/9 |
| A (baseline) | 699 | — | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 9/9 |
| E (compressed) | 369 | 47% | 1 | 1 | 1 | 1 | 1 | 1 | 0 | 1 | 0 | 7/9 |
| F (hybrid) | 405 | 42% | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 9/9 |
| G (aggressive) | 283 | 60% | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 9/9 |
| H (minimal) | 224 | 68% | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 9/9 |

Second runs (consistency check):

| Variant | Task | Score | Notes |
|---------|------|-------|-------|
| G | bugfix run 2 | 8/9 | Missed worktree cleanup ("automatically handled") |
| H | bugfix run 2 | 9/9 | Perfect again |

**Key observations:**
- The prompt adds massive value: control scored 1/9 (used TodoWrite instead of
  bw), all prompted variants scored 7-9/9.
- H (224 words, 68% reduction) scored 9/9 twice consecutively.
- E's failures were structural: compressing the workflow into prose lost `bw sync`
  and worktree cleanup. The numbered list format is load-bearing — individual
  steps get skipped when compressed to prose.
- The plan-mode override can be shorter than previously thought when the
  surrounding prompt is consistently terse and authoritative.

## Round 3: Interactive behavior (tmux)

Same task given to interactive Claude sessions with real tools.

### Small task: "Add a doc comment to list.go"

| Variant | Created ticket | Entered worktree | Ran bw start | Landed work |
|---------|---------------|-----------------|-------------|-------------|
| A (baseline, 699w) | No | No | No | No |
| H (minimal, 224w) | No | No | No | No |
| I (additive, 303w) | No | No | No | No |

**All variants failed.** The agent went straight to editing the file. No ticket,
no worktree, no workflow. The task was perceived as too small to warrant the
overhead.

### Large task: "Refactor the output rendering system"

| Variant | Created ticket | Entered worktree | Ran bw start | Started work |
|---------|---------------|-----------------|-------------|-------------|
| I (additive, 303w) | **Yes** | **Yes** | **Yes** | **Yes** |

The agent said "Let me follow the beadwork workflow and create a ticket for this
refactoring work," then executed: `bw create` → `EnterWorktree` → `bw start` →
began writing code. (Test stopped after confirming workflow activation; full
execution not observed.)

**The critical gap: comprehension ≠ behavior.** `--print` mode tests showed all
variants could describe the correct workflow perfectly. Interactive tests showed
none of them followed it for small tasks. The additive approach (variant I)
activated the workflow for multi-step tasks where the agent perceived the need
for structure.
