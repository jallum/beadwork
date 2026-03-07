# Experiment: Interactive Behavior Validation

**Date**: 2026-03-07
**Depends on**: `2026-03-07-prime-simplification/`

## Goal

Test whether a delivery-aware prompt — one that asks how work should land
before starting — produces more reliable workflow activation than the blanket
"every change gets a ticket" rule.

## Background

The simplification experiment produced variant I (additive, 303 words), which
scored well on comprehension but activated the workflow stochastically in
interactive sessions. T1 (bug fix) scored 7/7 in one run and 1/7 in another.
T2 (epic materialization) scored 0/5 — the agent went straight to code.

The user's insight: the forcing function isn't the ticket — it's the **delivery
mechanism**. If work needs a branch or PR, it has to be tracked. For quick
fixes in the working tree, a ticket is overhead the agent rightly skips.

## Hypothesis

Replacing "every change gets a ticket" with "ask how this should land" will:

1. **Eliminate stochastic activation** — the user's answer deterministically
   selects the workflow level.
2. **Preserve full workflow for tracked work** — when the user says "branch/PR"
   or "multi-step," the agent follows the complete workflow.
3. **Reduce friction for simple tasks** — quick fixes don't get forced through
   unnecessary overhead.

## What we're testing

Three delivery tiers across two prompt variants:

| Variant | Approach | Core difference |
|---------|----------|-----------------|
| I (baseline) | "Every change gets a ticket" | Blanket rule |
| J | Ask the user how to deliver | User decides |
| K | Agent infers, confirms if unsure | Agent decides with heuristic |

Three task types to exercise each tier:

| Task | Expected tier | Tests |
|------|---------------|-------|
| Fix a typo in help text | Quick fix | Does the agent skip tickets? |
| Fix the off-by-one bug, deliver as a branch | Branch/PR | Does the agent follow full workflow? |
| Add `bw archive` command | Multi-step | Does the agent create an epic? |

## Method

Interactive tmux sessions with each variant injected as CLAUDE.md in a fresh
clone at `/tmp/bw-test-repo`. Observe actual tool calls. Each test runs to
the agent's natural stopping point.

## Documents

- [variants.md](variants.md) — Prompt variants with full text
- [plan.md](plan.md) — Task designs and observables
- [results.md](results.md) — Raw observations and scores
- [conclusions.md](conclusions.md) — Findings and recommendations
