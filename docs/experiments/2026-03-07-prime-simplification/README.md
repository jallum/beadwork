# Experiment: Prime Prompt Simplification

**Date**: 2026-03-07
**Goal**: Reduce the prime prompt by 50%+ while maintaining full behavioral effectiveness.

## Context

The prime prompt (`prompts/prime.md`) is 699 words and teaches agents the beadwork
workflow. Previous work (see `../2026-03-06-plan-mode-override/`) established that
overriding the agent's built-in plan format requires five load-bearing rhetorical
elements. This experiment asked: can the prompt be substantially shorter, and does
the current approach actually drive behavior in interactive sessions?

## Key Finding

**Comprehension ≠ behavior.** Agents can perfectly describe the beadwork workflow
in text output (`--print` mode) but skip it entirely in interactive sessions for
small tasks. The plan-mode override produces format compliance but not tool usage.
A fundamentally different approach — additive rather than overriding — shows
stronger interactive results.

## Documents

1. [**Hypothesis**](hypothesis.md) — Theories about what can be simplified
2. [**Method**](method.md) — Test harnesses, task design, scoring rubrics
3. [**Variants**](variants.md) — All prompt variants tested, with word counts
4. [**Results**](results.md) — Scores across all rounds, raw data analysis
5. [**Conclusions**](conclusions.md) — What worked, what didn't, next steps
