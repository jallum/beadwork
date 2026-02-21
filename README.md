# Beadwork

[![CI](https://github.com/jallum/beadwork/actions/workflows/ci.yml/badge.svg)](https://github.com/jallum/beadwork/actions/workflows/ci.yml)
[![coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/jallum/beadwork/badges/coverage.json)](https://github.com/jallum/beadwork/actions/workflows/ci.yml)

Filesystem-native issue tracking for AI coding agents. Issues are JSON files, structure is directories, sync is git.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/jallum/beadwork/main/install.sh | sh
```

Or download a binary from [releases](https://github.com/jallum/beadwork/releases/latest). To update an existing install: `bw upgrade`.

## Quick Start

```bash
bw init                                           # initialize in any git repo
bw create "Fix auth bug" --type bug -p 1          # create an issue
bw ready                                          # list unblocked work
bw comments add bw-a1b2 "Fixed in latest deploy"  # add a comment
bw close bw-a1b2                                  # close it
bw sync                                           # push to remote
```

## Commands

**Working With Issues**
```
bw create <title> [flags]           Create an issue (--parent, --type, -p, --silent)
bw show <id>... [--only <sections>] [--json]  Show issue details with deps (aliases: view)
bw list [filters] [--json]          List issues (--grep, --all, --deferred)
bw update <id> [flags]              Update an issue (--parent to set/clear)
bw close <id> [--reason <r>]        Close an issue
bw reopen <id>                      Reopen a closed issue
bw delete <id> [--force]            Delete an issue (preview by default)
bw comments <id>                    List comments on an issue
bw comments add <id> <text>         Add a comment (--author)
bw label <id> +lab [-lab] ...       Add/remove labels
bw defer <id> <date>                Defer until a date
bw undefer <id>                     Restore a deferred issue
bw history <id> [--limit N]         Show commit history for an issue
```

**Finding Work**
```
bw ready [--json]              List unblocked issues
bw blocked [--json]            List issues waiting on dependencies
```

**Dependencies**
```
bw dep add <id> blocks <id>    Add a dependency
bw dep remove <id> blocks <id> Remove a dependency
```

**Sync & Data**
```
bw sync                        Fetch, rebase/replay, push
bw export [--status <s>]       Export issues as JSONL
bw import <file> [--dry-run]   Import issues from JSONL (use - for stdin)
```

**Setup & Config**
```
bw init [--prefix] [--force]   Initialize beadwork
bw config get|set|list         View/set config options
bw upgrade [--check] [--yes]   Check for / install binary updates
bw upgrade repo                Upgrade repo schema to latest version
bw onboard                     Print AGENTS.md snippet
bw prime                       Print workflow context for agents
```

## Design

### Storage

All data lives on a git orphan branch (`beadwork`), manipulated directly in the git object database via [go-git](https://github.com/go-git/go-git). Nothing touches your working tree or index.

Each issue is a JSON file. Structural relationships (status, labels, dependencies) are encoded as zero-byte marker files in a directory hierarchy:

```
issues/
  bw-a1b2.json
status/
  open/
    bw-a1b2          (0 bytes)
labels/
  bug/
    bw-a1b2          (0 bytes)
blocks/
  bw-a1b2/
    bw-c3d4          (0 bytes)
parent/
  bw-a1b2/
    bw-c3d4          (0 bytes)
```

Every listing query is a directory read. Parent-child relationships use the same marker pattern, with cycle detection preventing circular hierarchies. Two agents working on different issues never touch the same file.

### Sync

Every CLI operation commits with a structured message that doubles as a replayable intent log:

```
create bw-a1b2 p1 task "Fix auth bug"
close bw-a1b2 reason="completed"
link bw-a1b2 blocks bw-c3d4
delete bw-a1b2
comment bw-a1b2 "Fixed in latest deploy"
```

`bw sync` fetches, rebases, and pushes. If rebase conflicts, it replays intents from commit messages against the current remote state. No merge drivers, no lock files, no custom conflict resolution.

### Agent Integration

```bash
bw onboard   # prints an AGENTS.md snippet for your project
bw prime     # prints full workflow context + live project state
```

## Migrating Between Beads and Beadwork

Both tools use a shared JSONL interchange format, so migrating is an export/import round-trip. Issue IDs, dependencies, and parent-child relationships are preserved.

### Beads → Beadwork

```bash
bw init                           # initialize beadwork in the same repo
bd export | bw import -           # pipe directly from beads
bw sync                           # push the beadwork branch
```

### Beadwork → Beads

```bash
bw export > issues.jsonl          # export from beadwork
bd import -i issues.jsonl         # import into beads
```

Use `--status open` on export to migrate only open issues, or `--dry-run` on import to preview before committing. Duplicate IDs are skipped automatically.

### Field Mapping

| Beads | Beadwork | Notes |
|-------|----------|-------|
| `owner` | `assignee` | |
| `issue_type` | `type` | |
| `created_at` | `created` | |
| `dependencies` | `blocks` / `blocked_by` / `parent` | Flattened into separate fields |
| `comments` | `comments` | Text, author, timestamp preserved |
| — | `labels` | Beadwork-only; not present in beads |

## Requirements

- Git

## License

MIT
