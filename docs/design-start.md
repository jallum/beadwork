# Design: start.md

Design requirements for the start prompt (`prompts/start.md`).

This is the point-of-action briefing shown when an agent claims work with
`bw start`. It may be the ONLY beadwork context a sub-agent ever sees.

1. **Full briefing.** This template controls everything after the issue summary,
   description, and comments. It orchestrates display sections (children,
   deps) via bw commands and provides type-conditional starting and landing
   guidance.

2. **Be self-contained.** A sub-agent may see this with no other beadwork
   context. The steps must make sense without having read prime.

3. **Be concrete.** Prime teaches principles; this gives specific steps with
   the issue ID filled in. No room for interpretation.

4. **Adapt to project configuration.** Only show workflow steps (PR review,
   etc.) that are relevant to how this repo is configured.

5. **Adapt to issue type.** Epics are organizational containers — their work
   lives in the children, not in direct code changes. Tasks and bugs get
   scoped commit + landing steps.

6. **Stay brief.** The agent's attention is on the work ahead; the briefing
   should be scannable, not a wall of text.

7. **One ticket, one commit.** Agents batch multiple tickets into one commit,
   which muddles code review and makes interrupted work hard to untangle.
   The steps should make clear that landing is scoped to this ticket and
   happens before moving on to the next one.
