# Prompt System

Beadwork's prompt system teaches agents to use beadwork effectively. This
document covers what the system is trying to accomplish, how the prompts
relate to each other, and how to validate that changes are working.

Per-prompt design requirements live in dedicated docs:
- [`agents.md`](agents.md) — the boot loader
- [`prime.md`](prime.md) — the mental model
- [`start.md`](start.md) — the point-of-action briefing

## End Goals

An agent that has been through the prompt system should:

1. **Understand why beadwork exists.** Context loss is inevitable; durable
   state in git is the remedy. The agent should reach for beadwork not
   because it was told to, but because it understands that its built-in
   planning tools don't survive compaction.

2. **Materialize plans into beadwork.** Agents plan naturally in context —
   that thinking is valuable. But a plan that lives only in context dies at
   compaction. Before executing a multi-step plan, the agent should
   materialize it as an epic with children and dependencies. The plan is
   scratch; the tickets are what survive. `bw ready` is the execution loop.

3. **Know how to start safely.** Worktrees isolate agent work from the
   user's working tree. `bw start` delivers point-of-action instructions.
   Landing (commit, close, sync) is part of doing — not a separate step.

4. **Match workflow to delivery intent.** Not every change needs a ticket.
   The agent should ask how the user wants work delivered — quick fix (no
   ticket), branch/PR (ticket + worktree), or multi-step (epic). The user's
   answer determines the workflow level. Blanket rules like "every change
   gets a ticket" activate stochastically; the delivery question activates
   deterministically (see bw-u3u).

5. **Leave breadcrumbs.** Comments on issues serve double duty: notes to
   the agent's future self after compaction, and messages to collaborators
   in multi-agent setups.

6. **Delegate effectively.** Sub-agents don't inherit context. The
   orchestrator must include workflow steps in the handoff and verify
   the work landed.

## Prompt Architecture

Four prompts, each with a distinct role and delivery mechanism:

### `agents.md` — Boot loader

Injected into CLAUDE.md (always in context). Minimal: says "run `bw prime`"
and sets norms like "committing and closing are part of completing a task."
Its job is to get the agent to load the prime, not to teach beadwork.

**Why it's separate:** Always-in-context means every token is a tax on the
agent's attention budget for the entire session. This must be tiny.

See [`agents.md`](agents.md) for design requirements.

### `prime.md` — Mental model

Rendered dynamically by `bw prime`. Teaches the full system: why it exists,
how to plan, how to work, how to delegate. Includes live state (ready work,
WIP). This is the canonical reference — everything an agent needs to use
beadwork effectively in one read.

**Why it's separate:** Loaded on demand, so it can be thorough without
permanently consuming context. Dynamic rendering means it can include
current state (ready queue, WIP list) that makes it immediately actionable.

See [`prime.md`](prime.md) for design requirements.

### `start.md` — Point-of-action briefing

Rendered by `bw start <id>`. Delivers concrete landing instructions
specific to the ticket type and project configuration. Prime teaches
principles; start delivers the procedure at the moment the agent needs it.

**Why it's separate:** Procedure at point of action is more effective than
procedure in a mental-model document. A sub-agent may see *only* this
prompt — it must be self-contained.

See [`start.md`](start.md) for design requirements.

### `onboard.md` — One-time setup

Tells the user what to add to CLAUDE.md. Used by `bw init`, not by agents
directly. Exists to bootstrap the system — once agents.md is in place,
onboard.md's job is done.

## Experimentation

Prompt changes are hypotheses. The prompt system competes with agents'
built-in instructions (system prompts, plan mode templates, tool-specific
guidance) for influence over agent behavior. Changes must be tested against
actual agent behavior, not just read for plausibility.

### Build and test loop

1. **Edit** the prompt source in `prompts/`.
2. **Compile**: `go build -o ./bw-test ./cmd/bw`
3. **Install**: `ln -sf "$(pwd)/bw-test" ~/.local/bin/bw`
4. **Verify rendering**: `bw prime --x-render-as markdown` (or `tty`)
5. **Test against an agent** — give it a task that exercises the behavior
   you're trying to influence, and observe what it does.
6. **Restore**: `ln -sf /Users/j5n/.local/share/beadwork/bw-0.10.0 ~/.local/bin/bw`

### Testing with a second agent instance

Use tmux to spawn an isolated Claude instance:

```bash
tmux new-session -d -s test-prompt
tmux send-keys -t test-prompt 'unset CLAUDECODE && claude' Enter
```

Send tasks and observe behavior:
```bash
tmux send-keys -t test-prompt 'plan a multi-step refactor of the auth module' Enter
```

Capture and analyze output:
```bash
tmux capture-pane -t test-prompt -p -S -300 | grep -iE "(Epic|Steps|Sequencing|Context|Verification)"
```

**Caution:** Auto-approve loops can accidentally approve plan execution,
causing the test instance to modify source files. Use targeted approval
or review each prompt.

### What to test

| Behavior | How to observe |
|----------|----------------|
| Plans materialized as tickets | Give a multi-step task; check if `bw create` produces an epic with children |
| Tickets created before code changes | Give a fix task; check if `bw create` is called before editing |
| Worktrees used | Give any code task; check if EnterWorktree is invoked |
| Landing completed | Give a task; check if commit + close + sync happen |
| Sub-agent delegation includes workflow | Give a task requiring delegation; inspect the handoff prompt |

### `--print` vs interactive testing

`--print` mode measures **comprehension** — whether the agent can describe the
correct workflow. Interactive sessions (via tmux) measure **behavior** — whether
the agent actually follows it. These diverge: agents that score perfectly on
comprehension may skip the workflow entirely in interactive sessions, especially
for small tasks. Always validate with interactive tests before adopting a prompt
change. See bw-cm6 for detailed evidence.

### Interpreting results

A single test run is anecdotal. Agent behavior is stochastic — the same
prompt can produce different results across runs. When testing a prompt
change:

- **Run at least 2-3 trials** for the target behavior.
- **Vary the task** — a prompt that works for "refactor auth" may fail for
  "add a feature." Test across task types.
- **A/B test** — compare the new prompt against the old one with the same
  task to isolate the effect of the change.

### Documented experiments

Experiments are tracked as bw epics with the `experiment` label.
Use `bw list --label experiment` to find them. Key completed experiments:

- **bw-g6n** — Plan-mode override (superseded by additive approach)
- **bw-cm6** — Prime simplification (comprehension ≠ behavior)
- **bw-u3u** — Interactive validation (delivery-aware > blanket rules)
