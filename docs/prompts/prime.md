# prime.md

Design requirements for the prime prompt (`prompts/prime.md`).

For end goals, prompt architecture, and experimentation methodology,
see [`prompts.md`](prompts.md).

## Design requirements

1. **Combine principle with procedure.** Principles ("context loss is certain")
   set disposition but don't drive behavior at the point of action. Procedures
   (numbered workflow steps) drive behavior but feel arbitrary without
   rationale. Effective prompts pair a brief *why* with a concrete *what*.
   Tested: principle alone fails, procedure alone fails, both together work.

2. **Be additive, not overriding.** Agents arrive with built-in planning,
   formatting, and task management. Fighting these instincts requires heavy
   rhetorical force and still doesn't translate to interactive behavior.
   Instead, work *with* the agent's natural patterns and add a durability
   step: "plan however you want, then materialize as tickets before
   executing." Tested: the additive approach produced actual `bw create` →
   worktree → `bw start` execution in interactive sessions; the override
   approach produced format compliance in text output but not tool usage.

3. **Let the user decide the delivery level.** Blanket rules ("every change
   gets a ticket") activate stochastically — agents override them based on
   their own cost/benefit heuristic (7/7 in one run, 1/7 in the next).
   Instead, ask the user how they want work delivered: quick fix (no ticket),
   branch/PR (ticket + worktree), or multi-step (epic). The user's answer
   deterministically selects the workflow level. Tested: variant J asked
   and followed through correctly on 3/3 tests; variant I with blanket
   rules was stochastic. See `experiments/2026-03-07-interactive-validation/`.

4. **Use numbered lists for workflow.** When the workflow is a numbered
   checklist, agents walk through every step. When compressed to prose,
   individual steps get skipped. Tested: variant E compressed the 5-step
   workflow to a paragraph and lost `bw sync` and worktree cleanup;
   variants F/G/H kept the list and scored perfectly despite being shorter
   overall.

5. **Start from clean state.** A "check `git status`" instruction before
   starting work causes agents to notice and investigate dirty state.
   Without it, agents ignore uncommitted changes from previous sessions.
   The wording should direct agents to ask the user about leftover changes,
   not just "resolve" them (which agents interpret as "understand"). Tested:
   variant J2 checked git status as its first action. See
   `experiments/2026-03-07-interactive-validation/`.

6. **Teach worktree hygiene as part of the workflow.** Worktrees aren't
   a standalone concern — they're step 1 of the numbered workflow. When
   presented as inseparable from claiming work, agents follow through.
   When presented as a separate section, agents evaluate and skip.
   Sub-agents must always use worktrees for isolation regardless of
   delivery level — this prevents cross-contamination between concurrent
   agents.

7. **Stay compact.** Shorter is not just cheaper — it's more effective.
   224 words scored identically to 699 words on comprehension. 303 words
   (additive variant) produced better interactive behavior than 699 words
   (baseline). Less noise means less competition for attention.

8. **Keep state at the bottom.** The ready queue and WIP list make the
   prompt immediately actionable. Moving them above instructions caused
   total failure (0/7) — the agent fell back to default behavior entirely.
   State should follow instructions, not precede them.

9. **Adapt to project configuration.** Per-task conditionals (PR review,
   etc.) live in start.md and render at point of action. Prime shows the
   full mental model to all agents regardless of configuration.

10. **Be the canonical reference.** AGENTS.md is deliberately minimal — just
    a pointer to `bw prime`. This prompt is the single source of truth for
    how to use beadwork in this project.

11. **Land the work.** Prime establishes the principle (unfinished
    bookkeeping is invisible progress); `bw start` delivers the concrete
    steps via start.md. The numbered workflow list reinforces landing as
    step 4, not a separate concern.

12. **Teach delegation concisely.** When orchestrating sub-agents, the
    orchestrator must include workflow steps in the handoff. A compressed
    one-line delegation instruction works as well as a full paragraph —
    the key information is the sequence (worktree → start → work → comment
    → close) and the principle (they don't inherit your context).

13. **No implementation details or setup instructions.** Keep the focus on
    usage and mental model.

14. **Don't fight — augment.** The prime prompt competes with agents'
    built-in instructions (system prompts, plan mode templates). Overriding
    these requires escalating rhetorical force (bold, MUST, negations,
    consequences, override directives — all five required together) and
    still only wins format compliance, not behavioral change. The additive
    approach sidesteps the conflict entirely: let the agent plan in whatever
    format it wants, then add a materialization step. See
    `experiments/2026-03-07-prime-simplification/` for tested evidence.
