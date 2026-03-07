# Conclusions

## What the data shows

### 1. The prompt can be dramatically shorter without losing comprehension

H (224 words, 68% reduction) scored 9/9 on the execution workflow rubric twice
consecutively. The current prompt's length is not buying additional effectiveness
for workflow comprehension.

### 2. The numbered workflow list is load-bearing

Variant E compressed the 5-step workflow into a prose paragraph and lost two
behaviors (`bw sync`, worktree cleanup). Variants F, G, and H — all shorter
than E — kept the numbered list and scored 9/9. The list format functions as
a checklist the agent walks through; prose makes individual steps skippable.

### 3. "Brilliant and amnesiac" is dead weight for behavior

Variant C (direct opening) matched the baseline on every dimension. The literary
framing may set a disposition but doesn't drive measurable behavior.

### 4. Section order matters destructively

Variant B (state first) scored 0/7 — the only total failure across all
experiments. Moving ready/WIP before instructions disrupted the instructional
flow enough that the agent fell back to default behavior entirely.

### 5. Comprehension ≠ behavior

This is the most important finding. In `--print` mode, agents perfectly describe
the full beadwork workflow — ticket creation, worktrees, claiming, landing. In
interactive sessions with real tools, they skip the workflow entirely for small
tasks. The agent's cost/benefit heuristic treats small tasks as not worth the
overhead, regardless of what the prompt says.

### 6. Override < Additive

The plan-mode override (60 words of rhetorical force) produces format compliance
in `--print` mode but doesn't translate to interactive behavior. Variant I's
additive approach ("plan however you want, then materialize as tickets") produced
actual `bw create` → worktree → `bw start` execution in an interactive session
for a multi-step task — without any override language.

The difference: overriding fights the agent's instincts at the format level;
additive works with the agent's instincts and adds a materialization step.

## What remains uncertain

### Small-task activation

No variant — including the baseline — activated the workflow for a small task
("add a doc comment"). This may be correct behavior (trivial edits don't need
tickets) or it may be a gap in the prompt. "Before you touch code, create a
ticket" is in variant I but was ignored. Stronger activation language, or
accepting this as the natural threshold, are both viable paths.

### Additive approach completeness

Variant I was only tested interactively on one large task. It created a single
task ticket, not an epic with children — the agent may need to perceive more
complexity before it structures work as an epic. More trials across task sizes
and types are needed.

### Plan format in the additive approach

Variant I removes the plan-mode override. When the agent enters plan mode, will
it produce useful plans? The additive approach doesn't care about plan *format*
— it cares about plan *materialization*. But if the agent's plan is pure prose,
the materialization step becomes harder. The mermaid graph example in the epic
description may be sufficient guidance.

### Stochastic variance

Most results are single runs. Agent behavior varies across runs. H scored 9/9
twice on bugfix tasks; G scored 9/9 then 8/9. Before adopting any variant as
the new prompt, it needs multiple trials across task types.

## Recommended next steps

1. **Run variant I through more interactive trials** — multiple task sizes,
   planning tasks, delegation tasks. Confirm the additive approach holds up.

2. **Test small-task activation** — try stronger activation language in variant I
   ("Always create a ticket before editing code, even for one-line changes") and
   see if it changes interactive behavior for small tasks. If it doesn't, accept
   the threshold.

3. **Test plan materialization** — give variant I a multi-step task in plan mode.
   Does the agent produce a plan and then create an epic with children? Or does
   it stop at a single ticket?

4. **Decide on the small-task policy** — is "every change gets a ticket" actually
   the right policy, or should the prompt acknowledge a threshold? Fighting the
   agent's instincts here may be as futile as fighting the plan format.
