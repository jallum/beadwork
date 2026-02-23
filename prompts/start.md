<!--
Design requirements for this prompt:

This is the point-of-action briefing shown when an agent claims work with
`bw start`. It may be the ONLY beadwork context a sub-agent ever sees.

1. Full briefing. This template controls everything after the issue summary
   and description. It orchestrates display sections (children, deps,
   comments) via bw commands and provides type-conditional starting and
   landing guidance.

2. Be self-contained. A sub-agent may see this with no other beadwork
   context. The steps must make sense without having read prime.

3. Be concrete. Prime teaches principles; this gives specific steps with
   the issue ID filled in. No room for interpretation.

4. Adapt to project configuration. Only show workflow steps (PR review,
   etc.) that are relevant to how this repo is configured.

5. Adapt to issue type. Epics are organizational containers — their work
   lives in the children, not in direct code changes. Tasks and bugs get
   scoped commit + landing steps.

6. Stay brief. The agent's attention is on the work ahead; the briefing
   should be scannable, not a wall of text.

7. One ticket, one commit. Agents batch multiple tickets into one commit,
   which muddles code review and makes interrupted work hard to untangle.
   The steps should make clear that landing is scoped to this ticket and
   happens before moving on to the next one.
-->
<!-- IF type == epic -->
<!-- bw show {id} --only children -->
<!-- bw show {id} --only comments -->

STARTING THE WORK
<!-- IF workflow.review == pr -->
  - Push the branch for this worktree, and open a draft PR referencing it to preserve progress.
<!-- END -->
  - Work through this epic's children — use `bw ready` to find the next one.

LANDING THE WORK
  - Push the branch for this worktree, and convert it to ready for review.
  - Close the epic (`bw close {id}`) when all children are done
  - `bw sync`.
<!-- END -->
<!-- IF type == task -->
<!-- bw show {id} --only blockedby,unblocks -->
<!-- bw show {id} --only comments -->

<!-- IF workflow.review == pr -->
STARTING THE WORK
  If this will take multiple sessions, open a draft PR early to preserve progress.
<!-- END -->

LANDING THE WORK
  Land this ticket before starting the next one:
  - Commit only the changes for this ticket, referencing {id}
  - Close the ticket (`bw close {id}`); It will tell you if work is newly unblocked
<!-- END -->
<!-- IF type == bug -->
<!-- bw show {id} --only blockedby,unblocks -->
<!-- bw show {id} --only comments -->

<!-- IF workflow.review == pr -->
STARTING THE WORK
  If this will take multiple sessions, open a draft PR early to preserve progress.
<!-- END -->

LANDING THE WORK
  Land this ticket before starting the next one:
  - Commit only the changes for this ticket, referencing {id}
<!-- IF workflow.review == pr -->
  - Push the branch for this worktree, and open a PR referencing it. If a draft PR already exists, convert it to ready for review.
<!-- END -->
  - Close the ticket (`bw close {id}`); It will tell you if work is newly unblocked
  - `bw sync`.
<!-- END -->
