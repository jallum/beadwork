# Beadwork Workflow Context

> **Context Recovery**: Run `bw prime` after compaction, clear, or new session

## Session Close Protocol

**CRITICAL**: Before saying "done" or "complete", you MUST run this checklist:

```
[ ] 1. git status              (check what changed)
[ ] 2. git add <files>         (stage code changes)
[ ] 3. bw sync                 (commit beadwork changes)
[ ] 4. git commit -m "..."     (commit code)
[ ] 5. bw sync                 (commit any new beadwork changes)
[ ] 6. git push                (push to remote)
```

**NEVER skip this.** Work is not done until pushed.

## Core Rules
- **Default**: Use beadwork for ALL task tracking (`bw create`, `bw ready`, `bw close`)
- **Prohibited**: Do NOT use TodoWrite, TaskCreate, or markdown files for task tracking
- **Workflow**: Create bw issue BEFORE writing code, mark in_progress when starting
- Persistence you don't need beats lost context
- Git workflow: run `bw sync` at session end
- Session management: check `bw ready` for available work

## Essential Commands

### Finding Work
- `bw ready` - Show issues ready to work (no open blockers)
- `bw list --status open` - All open issues
- `bw list --status in_progress` - Your active work
- `bw show <id>` - Detailed issue view with dependencies

### Creating & Updating
- `bw create "Title" -p 2 -t task` - New issue
  - Priority: 1-5 (1=critical, 3=default, 5=backlog). NOT "high"/"medium"/"low"
  - Type: task, bug, epic
  - Also: `-a <assignee>`, `-d <description>`
- `bw update <id> --status in_progress` - Claim work
- `bw update <id> --assignee me` - Assign to someone
- `bw update <id> --title/--priority/-d` - Update fields
- `bw close <id>` - Mark complete
- `bw close <id> --reason "explanation"` - Close with reason
- `bw reopen <id>` - Reopen a closed issue
- `bw label <id> +bug +urgent -wontfix` - Add/remove labels

### Dependencies
- `bw link <blocker> blocks <blocked>` - Add dependency (blocker must close before blocked is ready)
- `bw unlink <blocker> blocks <blocked>` - Remove dependency
- `bw graph <id>` - Show dependency tree for an issue
- `bw graph --all` - Show all open issue dependencies

### Sync
- `bw sync` - Fetch, rebase/replay, push to remote

## Common Workflows

**Starting work:**
```
bw ready                               # Find available work
bw show <id>                           # Review issue details
bw update <id> --status in_progress    # Claim it
```

**Completing work:**
```
# Commit code FIRST, then close the ticket
git add <files> && git commit -m "..."
bw close <id> --reason "done"
bw sync
```

**Creating dependent work:**
```
bw create "Implement feature X" -t task -p 2
bw create "Write tests for X" -t task -p 2
bw link <feature-id> blocks <test-id>
```

## Notes
- `--json` on any read command for structured output
- IDs support prefix matching ("a1b2" matches "proj-a1b2")
- Statuses: open, in_progress, closed
