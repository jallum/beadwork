# Beadwork

Beadwork is a git-native work management tool built for AI coding agents. Issues are JSON files stored on a dedicated `beadwork` branch — no external service, no API keys, no network dependency. Sync is just `git push`.

## Why it exists

AI agents lose context constantly — compaction, session boundaries, crashes. Beadwork gives agents durable state that survives all of these. An agent can checkpoint progress, record decisions, and pick up exactly where it left off in a new session.

## How it works

`bw init` adds a `beadwork` branch to any git repo. Issues live there as JSON files with status, priority, dependencies, labels, comments, and parent/child relationships. The CLI (`bw`) is the only interface — agents use it the same way a human would.

Key commands: `bw create`, `bw ready` (find unblocked work), `bw start` (claim a task), `bw close`, `bw sync`. Every command supports `--help`.

## Repo structure

- `cmd/bw/` — CLI entry point and all subcommands
- `internal/` — core packages: `issue`, `repo`, `intent`, `template`, `treefs`, `wrap`
- `prompts/` — prompt templates embedded into the binary (`prime.md`, `start.md`, `agents.md`, `onboard.md`)
- `test/` — acceptance test harness

## Agent onboarding

When a project adopts beadwork, `bw onboard` prints a snippet (sourced from `prompts/agents.md`) for the user to add to their `AGENTS.md` or `CLAUDE.md`. That snippet points agents at `bw prime`, which delivers full workflow context dynamically.
