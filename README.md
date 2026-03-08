# Beadwork

[![CI](https://github.com/jallum/beadwork/actions/workflows/ci.yml/badge.svg)](https://github.com/jallum/beadwork/actions/workflows/ci.yml)
[![coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/jallum/beadwork/badges/coverage.json)](https://github.com/jallum/beadwork/actions/workflows/ci.yml)

Git-native work management for AI coding agents.

## Why

AI coding agents lose context constantly — compaction, session boundaries, crashes. Plans they build in-context die silently. The agent picks up where it left off with no memory of what it decided or why.

Beadwork gives agents durable state in git. Plans, progress, and decisions persist as files on a git branch. Nothing touches your working tree. Two agents working on the same repo never touch the same file. Sync is just git push.

## What It Looks Like

You talk to your agent normally. Beadwork works behind the scenes.

- _"Plan the auth module refactor"_
- _"What's next?"_
- _"Continue where we left off"_
- _"Use a team to do bw-xyz and bw-abc"_

## Community

Join the [Beadwork Discord](https://discord.gg/WCp4wuJwKe) for discussion, support, and updates.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/jallum/beadwork/main/install.sh | sh
```

Or download a binary from [releases](https://github.com/jallum/beadwork/releases/latest). To update an existing install: `bw upgrade`.

Coming from beads? see [docs/migration.md](docs/migration.md).

## Quick Start

```bash
bw init                                           # initialize in any git repo
bw onboard                                        # print bootstrap snippet for CLAUDE.md / GEMINI.md
bw create "Fix auth bug" --type bug -p 1          # create an issue
bw ready                                          # list unblocked work
bw comment bw-a1b2 "Fixed in latest deploy"       # add a comment
bw close bw-a1b2                                  # close it
bw sync                                           # push to remote
```

## Commands

**Managing Work**

```
bw create <title> [flags]           Create an issue (--parent, --type, -p, --silent)
bw show <id>... [--only <sections>] [--json]  Show issue details with deps (aliases: view)
bw list [filters] [--json]          List issues (--grep, --all, --deferred)
bw update <id> [flags]              Update an issue (--parent to set/clear)
bw close <id> [--reason <r>]        Close an issue
bw reopen <id>                      Reopen a closed issue
bw delete <id> [--force]            Delete an issue (preview by default)
bw comment <id> <text>              Add a comment (--author; use bw show to view)
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
bw onboard                     Print agent instructions snippet
bw prime                       Print workflow context for agents
```

## Agent Integration

`bw onboard` prints a snippet for your project's agent instructions file (CLAUDE.md, GEMINI.md, etc.). Once installed, agents automatically load workflow context via `bw prime` at the start of each session.

## Design

All data lives on a git orphan branch, manipulated directly in the object database via [go-git](https://github.com/go-git/go-git). Every operation is an atomic commit. Sync uses fetch-rebase-push with intent replay on conflict.

For storage layout and sync mechanics, see [docs/design.md](docs/design.md).

## License

MIT
