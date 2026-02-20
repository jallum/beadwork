# Beadwork

[![CI](https://github.com/jallum/beadwork/actions/workflows/ci.yml/badge.svg)](https://github.com/jallum/beadwork/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jallum/beadwork/graph/badge.svg)](https://codecov.io/gh/jallum/beadwork)

Filesystem-native issue tracking for AI coding agents. Issues are JSON files, structure is directories, sync is git.

## Install

Download a prebuilt binary from [releases](https://github.com/jallum/beadwork/releases/latest), or build from source:

```bash
go install github.com/j5n/beadwork/cmd/bw@latest
```

To update an existing install:

```bash
bw upgrade
```

## Quick Start

```bash
bw init                                    # initialize in any git repo
bw create "Fix auth bug" --type task -p 1  # create an issue
bw ready                                   # list unblocked work
bw close bw-a1b2                           # close it
bw sync                                    # push to remote
```

## Commands

```
bw init [--prefix] [--force]   Initialize beadwork
bw create <title> [flags]      Create an issue
bw show <id> [--json]          Show issue details
bw list [filters] [--json]     List issues
bw update <id> [flags]         Update an issue
bw close <id> [--reason <r>]   Close an issue
bw reopen <id>                 Reopen a closed issue
bw ready [--json]              List issues with no open blockers
bw label <id> +lab [-lab] ...  Add/remove labels
bw link <id> blocks <id>       Create dependency link
bw unlink <id> blocks <id>     Remove dependency link
bw graph <id>|--all [--json]   Dependency graph
bw config get|set|list         View/set config options
bw export [--status <s>]       Export issues as JSONL
bw import <file> [--dry-run]   Import issues from JSONL
bw sync                        Fetch, rebase/replay, push
bw onboard                     Print AGENTS.md snippet
bw prime                       Print workflow context for agents
bw upgrade [--check] [--yes]   Check for / install updates
```

## Design

### Storage

All data lives on a git orphan branch, checked out as a worktree inside `.git/beadwork/`. Nothing is visible in your working tree.

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
```

Every listing query is a directory read. Two agents working on different issues never touch the same file.

### Sync

Every CLI operation commits with a structured message that doubles as a replayable intent log:

```
create bw-a1b2 p1 task "Fix auth bug"
close bw-a1b2 reason="completed"
link bw-a1b2 blocks bw-c3d4
```

`bw sync` fetches, rebases, and pushes. If rebase conflicts, it replays intents from commit messages against the current remote state. No merge drivers, no lock files, no custom conflict resolution.

### Agent Integration

```bash
bw onboard   # prints an AGENTS.md snippet for your project
bw prime     # prints full workflow context + live project state
```

## Requirements

- Go 1.24+
- Git 2.5+ (worktree support)

## License

MIT
