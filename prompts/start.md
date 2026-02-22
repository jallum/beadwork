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
-->
- Commit referencing {id}
<!-- IF workflow.review == pr -->
- Push the branch and open a PR referencing {id}. Open it as a draft if work remains; as a regular PR if it's ready for review. If a draft PR already exists for this branch, convert it to ready for review when done.
<!-- END -->
- Close the ticket (`bw close {id}`); It will tell you if work is newly unblocked
- `bw sync`.
