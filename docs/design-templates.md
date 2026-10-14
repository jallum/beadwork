# Template System

Architecture for beadwork's Go `text/template` prompt rendering.

## Package

`internal/tmpl/tmpl.go` provides a thin wrapper around Go's `text/template`:

```go
func Execute(w io.Writer, name, src string, data any, bwFn func(args ...string) string) error
```

`Parse(name, src, bwFn)` returns a reusable `Template` with an
`Execute(w, data)` method. `bw start` loads and parses its prompt before claiming
an issue, so file-read and syntax errors leave the issue unclaimed. Execution
errors occur when rendering the briefing after the issue has been claimed.

## Prompt Overrides

`bw config set prompts.prime <path>` and `bw config set prompts.start <path>`
select custom files in place of the embedded templates. Loading belongs to the
CLI; `internal/tmpl` only parses and renders template text.

`bw config list` includes `prompts.prime=` and `prompts.start=` even when neither
key is stored. `bw config get <key>` returns the same empty default for these
options. Explicitly configured paths take precedence. These defaults are applied
by the CLI's config command for display only; reads do not write `.bwconfig` or
create commits. Unknown keys still return a `key not found` error from `config get`.

Relative paths are resolved using `git rev-parse --show-toplevel` from the command's
target working directory. This selects the current checkout rather than the main
repository's checkout when running in a linked worktree. Absolute paths are used
directly; shell variables and `~` in stored values are not expanded.

An unset or empty config value selects the embedded default. An empty file is a
valid replacement. Read, parse, and execution errors are reported with the config
key and file path; configured files never silently fall back to the default.
Only `prime` and the `start` briefing are configurable. `onboard` and its agent
instructions snippet continue to use the embedded prompts.

## `{{ bw }}` Template Function

The `bw` function runs beadwork commands and returns their output as a string
for inline embedding in the template. It captures output via TokenWriter; the
command's output writer resolves tokens to Markdown or terminal formatting.

Usage in templates:
```
{{ bw "ready" }}
{{ bw "list" "--status" "in_progress" }}
{{ bw "show" .ID "--only" "children" }}
```

The caller provides the `bwFn` closure which has access to the store and
command map. This keeps the tmpl package decoupled from the CLI.

## Template Data

### PrimeData (prime.md)

| Field | Source |
|-------|--------|
| `.Prefix` | `repo.ListConfig()["prefix"]` |
| `.WorktreeDirty` | Whether the current checkout has uncommitted changes |
| `.Git.Branch` | Current checkout's branch |
| `.Git.Dirty` | Whether the current checkout has uncommitted changes |
| `.Git.IsWorktree` | Whether the current checkout is a linked worktree |
| `.Git.LastCommit` | Latest commit's abbreviated hash and subject |
| `.OverdueCount` | Number of overdue issues |
| `.ExpiredDeferrals` | Markdown list of deferred issues now due for attention |

### StartData (start.md)

| Field | Source |
|-------|--------|
| `.ID` | Issue ID |
| `.Type` | Issue type (epic, task, bug) |
| `.Status` | Issue status |
| `.Parent` | Parent issue ID |
| `.WorkflowReview` | `repo.ListConfig()["workflow.review"]` |

### Example custom prime

Save this as `.beadwork/prompts/prime.md` on your code branch:

```markdown
# Project workflow

Work in the current checkout. Create a worktree only when the user requests one.
Record progress with `bw comment <id> "..."` and commit with the ticket ID.

## Work in progress
{{ bw "list" "--status" "in_progress" }}

## Available work
{{ bw "ready" "--no-context" }}
```

Enable it with `bw config set prompts.prime .beadwork/prompts/prime.md`.

## Blank Line Collapsing

`Execute()` collapses runs of 3+ consecutive newlines to 2 (one blank line).
This prevents conditional blocks that evaluate to empty from leaving visual
gaps in the output.

## Adding New Template Functions

Register additional functions in the `funcMap` within `Parse()`, or pass
them through the caller's closure. Keep the tmpl package generic — beadwork-
specific logic belongs in the caller (cmd/bw), not in the template package.
