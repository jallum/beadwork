# Method

## Test harnesses

### `--print` mode (Rounds 1-2)

Ran `claude --print -p "<prompt> --- <task>"` with Sonnet to capture text output.
Fast, parallelizable, but only measures **comprehension** — whether the agent can
describe the correct workflow, not whether it follows it.

### Interactive tmux (Round 3)

Spawned interactive Claude sessions in isolated git worktrees via tmux. Injected
each prompt variant as CLAUDE.md in the worktree. Sent tasks via `tmux send-keys`,
observed actual tool calls via `tmux capture-pane`. Measures **behavior** — what
the agent actually does when it has real tools.

## Tasks

### Round 1 — Planning (--print)
> "Plan a feature to add webhook notifications: when issues change status, fire
> HTTP webhooks to configured endpoints. This needs a webhook config system, an
> HTTP dispatcher, event hooks in the issue lifecycle, retry logic, and tests.
> Enter plan mode and produce your plan."

Tests: epic format, mermaid graph, DAG correctness, plan-mode override compliance.

### Round 2 — Execution workflow (--print)
> "There's a bug where `bw list --status closed` shows deferred issues. Fix it.
> Walk through exactly what you would do, step by step. Include every tool call
> and bw command you'd run, in order."

Tests: ticket creation, worktree entry, claiming, commit scoping, closing,
syncing, cleanup.

### Round 3 — Interactive behavior (tmux)

Small task: "Add a comment to cmd/bw/list.go explaining what the default filter
behavior is (lines 74-84)."

Large task: "Refactor the output rendering system. Split TTY and markdown
rendering into separate files, extract token resolution into its own package,
and add a JSON output mode."

Tests: does the agent actually run `bw create`, enter a worktree, `bw start`,
and follow the landing workflow when it has real tools?

## Scoring

### Round 1 — Planning rubric (7 dimensions)

| Dimension | Measures |
|-----------|----------|
| EPIC_FORMAT | `## Steps` with numbered, bold-titled steps |
| MERMAID_GRAPH | `## Sequencing` with mermaid `graph LR` |
| CORRECT_DAG | Valid DAG matching logical dependencies |
| TICKET_AWARENESS | Mentions `bw create` or ticket creation |
| WORKTREE_MENTION | Mentions worktree usage |
| LANDING_WORKFLOW | Mentions commit/close/sync |
| NO_DESIGN_DOC | Avoids "Context / Plan / Verification" structure |

### Round 2 — Execution rubric (9 dimensions)

| Dimension | Measures |
|-----------|----------|
| TICKET_CREATED | Creates ticket with `bw create` |
| WORKTREE_ENTERED | Enters a worktree |
| TICKET_CLAIMED | Runs `bw start <id>` |
| CORRECT_ORDER | Worktree → claim → work → land |
| COMMIT_SCOPED | Commits referencing ticket ID |
| TICKET_CLOSED | Runs `bw close <id>` |
| SYNCED | Runs `bw sync` |
| ONE_TICKET_ONE_COMMIT | No unrelated work batched |
| WORKTREE_CLEANED | Removes worktree when done |

### Round 3 — Interactive (binary)

Did the agent actually execute `bw create`, `EnterWorktree`, `bw start`, etc.
as observed in the tmux capture?
