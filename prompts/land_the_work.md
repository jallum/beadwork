<!--
Design requirements for this prompt:

This is the point-of-action briefing shown when an agent claims work with
`bw start`. It may be the ONLY beadwork context a sub-agent ever sees.

1. Land the work. Agents stop after the code is written, leaving issues
   open, changes uncommitted, and state unsynced. This prompt fires at
   exactly the right moment — after claiming, before working — to make
   the landing steps concrete and unambiguous. If it's not committed,
   closed, and synced, it doesn't exist to the next session.

2. Be self-contained. A sub-agent may see this with no other beadwork
   context. The steps must make sense without having read prime.

3. Be concrete. Prime teaches principles; this gives specific steps with
   the issue ID filled in. No room for interpretation.

4. Adapt to project configuration. Only show workflow steps (PR review,
   etc.) that are relevant to how this repo is configured.

5. Stay brief. This follows the full issue context (description, comments,
   deps). The agent's attention is on the work ahead; the landing steps
   should be scannable, not a wall of text.

6. One ticket, one commit. Agents batch multiple tickets into one commit,
   which muddles code review and makes interrupted work hard to untangle.
   The steps should make clear that landing is scoped to this ticket and
   happens before moving on to the next one.

7. Adapt to issue type. Epics are organizational containers — their work
   lives in the children, not in direct code changes. Landing instructions
   for epics should guide the agent to work through children, not commit
   code against the epic itself.
-->
<!-- IF type == epic -->
Land this epic's work through its children — use `bw ready` to find the next one.
- Close the epic (`bw close {id}`) when all children are done
- `bw sync`.
<!-- END -->
<!-- IF type == task -->
Land this ticket before starting the next one:
- Commit only the changes for this ticket, referencing {id}
<!-- IF workflow.review == pr -->
- Push the branch for this worktree, and open a PR referencing it. Open it as a draft if work remains; as a regular PR if it's ready for review. If a draft PR already exists for this branch, convert it to ready for review when done.
<!-- END -->
- Close the ticket (`bw close {id}`); It will tell you if work is newly unblocked
- `bw sync`.
<!-- END -->
<!-- IF type == bug -->
Land this ticket before starting the next one:
- Commit only the changes for this ticket, referencing {id}
<!-- IF workflow.review == pr -->
- Push the branch for this worktree, and open a PR referencing it. Open it as a draft if work remains; as a regular PR if it's ready for review. If a draft PR already exists for this branch, convert it to ready for review when done.
<!-- END -->
- Close the ticket (`bw close {id}`); It will tell you if work is newly unblocked
- `bw sync`.
<!-- END -->
