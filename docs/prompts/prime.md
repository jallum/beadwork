# prime.md

Design requirements for the prime prompt (`prompts/prime.md`).

For end goals, prompt architecture, and experimentation methodology,
see [`prompts.md`](prompts.md).

## Design requirements

1. **Teach principles, not procedures.** Explain the system — why it exists, how
   it works, what good usage looks like — and trust the agent to apply that
   understanding. Favor describing consequences over issuing commands.

2. **Position as augmentation.** Agents already have built-in planning and task
   management. Those work fine for small, tactical work. Beadwork doesn't
   replace them — it gives plans and progress durability that survives context
   loss. An in-memory todo list is one compaction away from gone; a bead isn't.

3. **Build confidence for larger work.** Agents are often hesitant to attempt
   ambitious, multi-file changes because they sense their own context limits.
   Beadwork makes that work safe to attempt — progress is checkpointed, state
   is recoverable, and losing context doesn't mean losing the plot.

4. **Plans belong in beadwork, not just in context.** Agents naturally plan in
   their context window. That's fine for understanding work, but a plan that
   lives only in context is lost at compaction. When work has multiple discrete
   steps — especially steps that could each be completed independently — the
   plan should be materialized as issues. An epic with children *is* the plan,
   and `bw ready` *is* the execution loop.

5. **Teach worktree hygiene as part of starting work.** Agents treat worktrees
   as optional hygiene — a separate concern they can evaluate independently.
   But the main working tree belongs to the user; agent work belongs in a
   worktree. Presenting this as inseparable from claiming work (not a
   standalone section) makes the connection harder to skip.

6. **Frame beadwork as shared state.** In multi-agent setups, beadwork is the
   durable communication layer between workers. Comments and issues serve
   double duty — breadcrumbs for your future self (surviving compaction) and
   messages to collaborators.

7. **Stay compact.** This goes into an agent's context window. Every unnecessary
   sentence is a tax on the agent's attention budget. Dense, scannable, no
   filler.

8. **Adapt to project configuration.** Per-task conditionals (PR review, etc.)
   now live in start.md and render at point of action. Prime shows the full
   mental model to all agents regardless of configuration.

9. **Be the canonical reference.** AGENTS.md is deliberately minimal — just a
   pointer to `bw prime`. This prompt is the single source of truth for how
   to use beadwork in this project.

10. **Land the work.** Prime establishes the principle (unfinished bookkeeping
    is invisible progress); `bw start` delivers the concrete steps via
    start.md. Prime should reinforce that landing matters without
    duplicating the procedure.

11. **Every task gets a ticket.** Agents skip ticket creation for small tasks,
    treating it as overhead. But tickets are cheap and capture intent —
    why a change was made, not just what changed. Commit messages record
    the what; tickets record the why. Without them, release notes and
    changelogs require reverse-engineering intent from diffs.

12. **No implementation details or setup instructions.** Keep the focus on usage
    and mental model.

13. **Teach delegation.** When orchestrating sub-agents, the orchestrator has
    beadwork context but the workers don't. The prompt should make clear
    that delegated tasks must include the workflow steps — claim, do, land
    — or the workers will skip them. The orchestrator is responsible for
    including the workflow and verifying the work landed.

14. **Win instruction conflicts.** The prime prompt is not the only voice
    the agent hears. Built-in tools (plan mode, task management) arrive
    with their own system-level instructions — specific formats, procedures,
    and templates that contradict beadwork's guidance. Principles alone
    don't win these conflicts; the agent defaults to the more specific,
    more procedural, higher-authority instruction. When beadwork's way
    genuinely matters (e.g., plan format), the prompt must combine
    principle (why) with enough procedural force (what, and permission to
    override) to compete. See the Experimentation section of
    [`prompts.md`](prompts.md) for tested evidence of what works.
