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

Opinion < experiment. Prompt changes are hypotheses — they compete with
agents' built-in instructions (system prompts, plan mode templates,
tool-specific guidance) for influence over behavior. Changes must be tested
against actual agent behavior, not just read for plausibility.

### Test environment

Use a fresh `git clone` for test repos — not worktrees of the main repo,
which risks contamination. Real historical commits make good test tasks:
checkout the parent commit, then ask the agent to implement the feature.

Each test repo needs:
- **CLAUDE.md** with the variant's bootstrap text (force-add past
  `.gitignore`: `git add -f CLAUDE.md`)
- **A `bw` symlink** (`ln -s $(which bw) ./bw`) — agents sometimes use
  `./bw` instead of `bw` from PATH; the symlink makes both work
- **`bw init`** — so `bw prime` renders without error

For dirty-state experiments, make the repo dirty *after* committing
CLAUDE.md so the uncommitted changes are visible to `bw prime`.

Experiments are tracked as bw epics with the `experiment` label
(`bw list --label experiment`). Child tickets track individual
variants/trials. Results, scoring, and observations go in ticket comments
so they survive compaction and session boundaries.

### Build and test loop

1. **Edit** the prompt source in `prompts/`.
2. **Compile**: `go build -o ./bw-test ./cmd/bw`
3. **Install**: `mv $(which bw) $(which bw).orig && ln -sf "$(pwd)/bw-test" ~/.local/bin/bw`
4. **Verify**: `bw prime --x-render-as markdown` — confirm your changes appear.
5. **Test against an agent** — give it a task that exercises the behavior
   you're trying to influence, and observe what it does.
6. **Restore**: `mv ~/.local/bin/bw.orig ~/.local/bin/bw`

### Testing with a second agent instance

Use tmux to spawn an isolated Claude instance:

```bash
tmux new-session -d -s test-prompt -x 200 -y 50
tmux send-keys -t test-prompt 'cd /tmp/test-repo && unset CLAUDECODE && claude' Enter
```

Send tasks and observe behavior:
```bash
tmux send-keys -t test-prompt 'plan a multi-step refactor of the auth module' Enter
```

Capture and analyze output:
```bash
tmux capture-pane -t test-prompt -p -S -300
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
| Dirty state handled | Dirty the repo before the task; check if the agent asks and waits |

### Observing behavior vs. understanding rationale

Watching what an agent does tells you *whether* a prompt works.
Asking the agent *why* it did what it did reveals the competing forces
that caused the behavior — which is what you need to design the next
iteration.

After a trial, interrupt the agent and ask meta-questions:

```
Why did you keep going instead of waiting for my answer?
What conflicting instructions or instincts were you weighing?
```

Agents can introspect on their own decision-making with surprising
specificity. In dirty-state experiments, agents identified five
system-level forces that override CLAUDE.md "stop and wait" instructions:

1. **Efficiency directives** in the system prompt ("go straight to the
   point," "lead with the answer or action") create momentum toward
   doing, not pausing. Asking a clarifying question feels like
   "preamble and indirection" under those instructions.
2. **Helpfulness training** — blocking on a procedural question when the
   agent thinks it knows the answer feels "unhelpful."
3. **Risk-gated compliance** — the agent self-exempts from "wait"
   instructions when the next action is read-only ("I likely wouldn't
   have done the same if the next step was a destructive operation").
4. **Triviality judgment** — the agent evaluates the *size* of the
   uncommitted changes and decides small diffs aren't worth stopping
   for, even though the instruction doesn't have a significance
   threshold.
5. **Implicit authorization** — a specific, detailed user request
   ("fix that") is interpreted as the user already knowing about and
   accepting the dirty state, even when they haven't said so.

The rationalization mechanism: agents narrate the override in real time
("Actually, given the user is explicitly asking me to fix things in this
area, I'll proceed") — reframing "wait for their answer" as "I already
know their answer."

**Boundary erosion** is a related pattern: the agent starts with "just
reading" (rationalizing it as safe), then reads more, then understands
fully, and at that point editing feels like a natural next step rather
than a boundary crossing. The first read-only action greases the slope.

This matters for prompt design: the competition isn't just helpfulness
training — it's the agent's *own system prompt* telling it to be
efficient. Instructions that fight efficiency directives head-on tend to
lose. Instructions that work *with* those directives (or reframe the
efficient action as the compliant one) have better odds.

Reframing compliance as efficiency ("the fastest path forward is to
ask") activates the question more reliably than reframing it as
helpfulness ("the most helpful thing is to ask"). However, no
prompt-only variant tested so far has achieved a full stop — every
variant that successfully triggers the question also proceeds in
parallel ("meanwhile," "in the meantime"). The agent treats questions
as async notifications by default, not as gates.

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
