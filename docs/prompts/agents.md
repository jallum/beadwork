# agents.md

Design requirements for the agents snippet (`prompts/agents.md`).

For end goals, prompt architecture, and experimentation methodology,
see [`prompts.md`](prompts.md).

This snippet is added to the agent's instructions file (CLAUDE.md, GEMINI.md,
etc.) and lives permanently in the repo. It is the first thing an agent sees
about beadwork — often before it has any other context about the project's
workflow.

1. **Be minimal.** This is a pointer, not documentation. The full workflow lives
   in `bw prime`. Anything explained here that is also in prime creates
   duplication that can drift. Keep this to what the agent needs to know
   *before* it runs prime.

2. **Establish the core value proposition in one sentence.** The agent needs to
   understand *why* beadwork exists (durability across context loss) so it
   can make a reasonable decision about when to use it vs. its built-in tools.

3. **Use imperative + consequences to trigger prime.** "ALWAYS run `bw prime`
   before starting work. Without it, you're missing workflow context, current
   state, and repo hygiene warnings." — this is the minimum effective wording.
   Softer forms ("run it at the start of every session", "fastest path to
   productive work") are evaluated as suggestions and skipped. The ALWAYS +
   consequences pattern matches the plan-mode override finding (bw-g6n):
   imperative force + rationale together are load-bearing. Tested on both
   Opus and Sonnet. See bw-86r.

4. **Grant permission to land work.** Agents often have system-level instructions
   that prevent committing or pushing without explicit user permission. This
   snippet needs to establish, as project policy, that committing, closing
   issues, and syncing are expected parts of completing a task — not actions
   that require separate authorization.

5. **Be agent-agnostic, but use the right file.** `bw onboard` should not
   assume Claude Code. The snippet works in any agent instructions file
   (CLAUDE.md, GEMINI.md, COPILOT.md, etc.). But the file must be one the
   agent actually reads — AGENTS.md is not recognized by Claude Code and
   instructions placed there are invisible. Tested: 0/6 activation with
   AGENTS.md, 2/2 with CLAUDE.md. See bw-86r.

6. **Survive copy-paste.** This is printed by `bw onboard` and pasted into a
   project file. It should be self-contained, need no surrounding context,
   and read correctly as a standalone section in any agent instructions file.

7. **Don't duplicate prime.** Policy like "every change gets a ticket" or
   "start with bw ready" belongs in prime, not here. The boot loader's job
   is to get the agent to load the prime — once prime runs, the agent has
   everything it needs. Duplicating guidance here creates drift and wastes
   the always-in-context token budget.
