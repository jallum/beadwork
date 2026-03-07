{{/* See docs/prompts/start.md */}}
{{ if eq .Type "epic" }}{{ bw "show" .ID "--only" "children" }}

## STARTING THE WORK
{{ if eq .WorkflowReview "pr" }}  - Push the branch for this worktree, and open a draft PR (`{{ .ID }}: <title>`) to preserve progress.
{{ end }}  - Work through this epic's children — use `bw ready` to find the next one.

## LANDING THE WORK
  - Push the branch for this worktree, and convert it to ready for review.
  - Close the epic (`bw close {{ .ID }}`) when all children are done
  - `bw sync`.
{{ end }}{{ if or (eq .Type "task") (eq .Type "bug") }}{{ bw "show" .ID "--only" "blockedby,unblocks" }}
{{ if eq .WorkflowReview "pr" }}
## STARTING THE WORK
  If this will take multiple sessions, open a draft PR early to preserve progress.
{{ end }}
## LANDING THE WORK
  Land this ticket before starting the next one:
  - Commit only the changes for this ticket, referencing {{ .ID }}
{{ if and (eq .Type "bug") (eq .WorkflowReview "pr") }}  - Push the branch for this worktree, and open a PR referencing it. If a draft PR already exists, convert it to ready for review.
{{ end }}  - Close the ticket (`bw close {{ .ID }}`); It will tell you if work is newly unblocked
{{ if eq .Type "bug" }}  - `bw sync`.
{{ end }}{{ end }}
