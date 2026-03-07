# Conclusions

## Primary Finding: Delivery-Aware > Blanket Rules

The "every change gets a ticket" approach (variant I) produces stochastic
workflow activation — 7/7 in one run, 1/7 in the next, 0/5 for epics. The
agent's cost/benefit heuristic overrides the prompt unpredictably.

The "how should this land?" approach (variant J) produces deterministic
activation by moving the decision to the user:

| Scenario | Variant I | Variant J |
|----------|-----------|-----------|
| Quick fix | Sometimes skips workflow | Asks → user says "just fix it" → no overhead |
| Branch work | Stochastic (50%) | Asks → user says "branch" → full workflow |
| Explicit cue | N/A | Recognizes "as a branch" without asking |

The forcing function shifts from the prompt (unreliable) to the user's
answer (deterministic).

## Secondary Finding: Hygiene Checks Work

Adding "check `git status` before starting" causes the agent to notice and
investigate dirty state. Without it, agents ignore uncommitted changes. The
wording matters: "resolve" was interpreted as "understand" rather than "act
on" — should be "ask the user what to do with leftover changes."

## Tertiary Finding: Repo State Affects Activation

A polluted repo (leftover files from previous work) caused variant J to skip
the delivery question on one run. Clean repo → asked. This reinforces the
hygiene check: the agent should start from a known-clean state.

## What to Change in the Prompt

1. Replace "Every Change Gets a Ticket" with "How Should This Land?" and
   three delivery tiers (quick fix / branch / multi-step).
2. Add a hygiene check: "Check `git status` — ask the user about leftover
   changes before starting new work."
3. Strengthen sub-agent isolation: "Sub-agents always work in worktrees."
4. Keep everything else from variant I (plans section, workflow list,
   delegation, breadcrumbs).

## Open Questions

- **Epic materialization**: Not tested with variant J. The multi-step tier
  exists in the prompt but hasn't been validated interactively.
- **Delegation with worktree isolation**: Not tested. The prompt says
  "sub-agents always work in worktrees" but we haven't observed whether
  this actually happens.
- **Repeat reliability**: Each variant J test ran once. The stochastic
  problem may still exist at lower frequency — more runs would increase
  confidence.
