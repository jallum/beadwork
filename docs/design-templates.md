# Design: Template System

Architecture for beadwork's Go `text/template` prompt rendering.

## Package

`internal/tmpl/tmpl.go` provides a thin wrapper around Go's `text/template`:

```go
func Execute(w io.Writer, name, src string, data any, bwFn func(args ...string) string) error
```

## `{{ bw }}` Template Function

The `bw` function runs beadwork commands and returns their output as a string
for inline embedding in the template. It captures output via PlainWriter
(ResolveMarkdown) so the result is clean markdown, not ANSI-colored.

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

### StartData (start.md)

| Field | Source |
|-------|--------|
| `.ID` | Issue ID |
| `.Type` | Issue type (epic, task, bug) |
| `.Status` | Issue status |
| `.Parent` | Parent issue ID |
| `.WorkflowReview` | `repo.ListConfig()["workflow.review"]` |

## Blank Line Collapsing

`Execute()` collapses runs of 3+ consecutive newlines to 2 (one blank line).
This prevents conditional blocks that evaluate to empty from leaving visual
gaps in the output.

## Adding New Template Functions

Register additional functions in the `funcMap` within `Execute()`, or pass
them through the caller's closure. Keep the tmpl package generic — beadwork-
specific logic belongs in the caller (cmd/bw), not in the template package.
