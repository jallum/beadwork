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

### Custom prompts

`bw config list` always includes the prompt options, even before you set them:

```text
prompts.prime=
prompts.start=
```

| Option | Controls | Default |
|--------|----------|---------|
| `prompts.prime` | Workflow context printed by `bw prime` | Empty: built-in `prime.md` |
| `prompts.start` | Briefing printed by `bw start <id>` | Empty: built-in `start.md` |

Replace the built-in workflow context or issue briefing with your own Markdown
files:

```sh
bw config set prompts.prime .beadwork/prompts/prime.md
bw config set prompts.start .beadwork/prompts/start.md
bw config get prompts.prime
```

Relative paths are resolved from the current checkout's repository root, including
when invoked from a subdirectory, through `-C`, or in a worktree. Absolute paths
are also supported. Settings live on the `beadwork` branch and sync with issues;
commit the prompt files on your code branch so they are available in each checkout.

Custom files replace the entire selected prompt and support the same Go template
variables and `{{ bw ... }}` function as the built-in templates. `bw start` still
prints the issue summary, description, and comments before its custom briefing.
`bw start --json` does not load or render a prompt.

Unset settings use the built-in prompts. Set a value to an empty string to restore
the default, for example `bw config set prompts.prime ""`. An empty prompt file
suppresses that prompt. Missing files and invalid templates return errors instead
of falling back to the built-in prompt. `bw config get` returns an empty value for
an unset prompt option. Listing or reading defaults does not save them to the
`beadwork` branch.

See [the template reference](docs/design-templates.md) for available variables and
an example. Custom prompts are maintained by your project and do not automatically
receive changes to the built-in templates.

For labeled local builds, such as `bw 0.13.3+xpmatteo`, see the
[custom build instructions](CONTRIBUTING.md).

## Design

All data lives on a git orphan branch, manipulated directly in the object database via [go-git](https://github.com/go-git/go-git). Every operation is an atomic commit. Sync uses fetch-rebase-push with intent replay on conflict.

For storage layout and sync mechanics, see [docs/design.md](docs/design.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the local dev loop and `make` targets that mirror CI.

## License

MIT
