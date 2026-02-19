# Beadwork: A Filesystem-Native Agent Memory System

## Project Proposal

### Problem

Beads is the leading coding agent memory system, but its recent migration from SQLite+JSONL to Dolt has introduced significant complexity and instability for single-agent and small-team workflows. The current architecture requires an embedded database engine, custom sync pipelines, lock file management, daemon processes, federation protocols, and a diagnostic system (`bd doctor`) that frequently cannot fix the issues it identifies.

The root cause is architectural: storing issues in a single database (first SQLite, then Dolt) created merge conflict problems that required increasingly complex solutions — JSONL export/import, custom merge drivers, tombstone systems, and eventually a full version-controlled SQL database. Each layer of complexity exists to solve a problem created by the previous layer.

### Core Insight

If each issue is stored as its own file, most of this complexity vanishes. Git already provides versioning, replication, history, collaboration, and transactional semantics. The job of the tool is to manipulate files and let git do everything else.

### Design

#### Filesystem as Database

The data model has two layers:

1. **Issue files** (JSON) — the complete, self-contained record of an issue, including redundant copies of structural data for fast single-issue reads.
2. **Marker files** (0-byte) — directory structure that encodes status, labels, and relationships for fast listing and filtering queries.

The marker files are authoritative for structural queries. The JSON files include redundant structural fields so that `bw show <id>` is a single file read.

```
issues/
  canon-a1b2.json
  canon-c3d4.json
  epic-e5f6.json

status/
  open/
    canon-a1b2           (0 bytes)
    epic-e5f6            (0 bytes)
  in_progress/
    canon-c3d4           (0 bytes)
  closed/

labels/
  bug/
    canon-a1b2           (0 bytes)
  frontend/
    canon-a1b2           (0 bytes)
    canon-c3d4           (0 bytes)

blocks/
  canon-a1b2/
    canon-c3d4           (0 bytes)

parent/
  epic-e5f6/
    canon-a1b2           (0 bytes)
    canon-c3d4           (0 bytes)
```

Every structural listing is a directory read:

- **Issues by status:** `ls status/open/`
- **Issues with label:** `ls labels/bug/`
- **What does A block?** `ls blocks/canon-a1b2/`
- **Children of an epic:** `ls parent/epic-e5f6/`
- **All labels:** `ls labels/`

Every structural mutation creates or removes marker files, and updates the redundant fields in the affected JSON file(s) — all in a single commit.

Two agents operating on different issues never touch the same file.

#### Issue Files

An issue file is pretty-printed JSON with alphabetically sorted keys, one field per line. It contains both content and redundant structural metadata for self-contained reads.

```json
{
  "assignee": "",
  "blocked_by": ["canon-a1b2"],
  "blocks": [],
  "created": "2026-02-19T10:00:00Z",
  "description": "Deploy the new auth flow to production",
  "id": "canon-c3d4",
  "labels": ["frontend"],
  "parent": "epic-e5f6",
  "priority": 2,
  "status": "in_progress",
  "title": "Deploy new auth flow",
  "type": "task"
}
```

The `status`, `labels`, `blocks`, `blocked_by`, and `parent` fields are redundant with the directory structure. They exist so that `bw show <id>` reads one file and has everything. On write, both the JSON file and the marker files are updated in the same commit. On sync conflict, intent replay regenerates both from scratch.

#### Issue IDs

IDs are a configurable prefix plus 3–5 random characters, chosen to not collide with any existing ID. `bw init --prefix canon` sets the prefix for the project (default derived from repo name).

If a sync pulls in a remote issue whose ID collides with a locally-created issue, the local issue gets a new ID. The intent log makes this safe — the local intent is replayed with a fresh ID against the current state.

#### Statuses

Three statuses, encoded by directory location:

- **open** — default for new issues
- **in_progress** — actively being worked
- **closed** — done

There is no `blocked` status. Whether an issue is blocked is determined by checking its `blocked_by` field (or scanning `blocks/` directories for reverse lookups). `bw ready` lists open issues whose blockers are all closed.

#### Orphan Branch Isolation

All beadwork data lives on a git orphan branch — always. This is not configurable. The orphan branch has no shared history with the code. No issue data ever appears on main, feature branches, or PRs. There is no option to put it elsewhere because every alternative leads to the same pollution and sync problems that plagued Beads.

`bw init` detects what already exists and does the right thing:

- **Remote branch exists** (collaborator already set up beadwork): just creates the local worktree.
- **No branch exists** (new project): creates the orphan branch, then the worktree.

```bash
# bw init does one of:

# Path A: branch already exists on remote
git fetch origin beadwork
git worktree add .git/beadwork beadwork

# Path B: new project, no branch yet
git checkout --orphan beadwork
git rm -rf .
git commit --allow-empty -m "init beadwork"
git checkout -
git worktree add .git/beadwork beadwork
```

No wizard. No prompts. No role selection. Look at what exists, do the right thing.

The CLI reads and writes to `.git/beadwork/`. Since it lives inside `.git/`, there's nothing to `.gitignore`, no dotfile in the project root, no visible trace in the working tree. It simply doesn't exist from the user's perspective.

Collaboration is just git:

```bash
git push origin beadwork      # share your issues
git fetch origin beadwork     # pull others' issues
```

#### Commit Messages as Intent Log

Every commit on the beadwork branch uses a structured message that serves two purposes: audit trail and replay log.

```
create canon-a1b2 p1 task "Auth token refresh bug"
close canon-a1b2 reason="completed"
update canon-a1b2 status=in_progress assignee=agent-1
link canon-a1b2 blocks canon-c3d4
label canon-a1b2 +bug +frontend
parent epic-e5f6 add canon-a1b2
```

As an audit trail, git log gives you free event-sourced history:

```bash
git log --grep="^create.*p1" --oneline     # all P1s ever created
git log --author="agent-1" --oneline        # everything agent-1 did
git log -- issues/canon-a1b2.json --oneline # full history of one issue
```

As a replay log, these messages are what the sync mechanism uses to re-apply operations when a clean rebase isn't possible. The messages must be machine-parseable and describe idempotent operations. Files are current state. Commits are the journal.

#### Optimistic Concurrency with Offline Support

The system works fully offline. Operations commit locally with structured messages that double as a replayable intent log.

**Local work (online or offline):**

```
1. read local state
2. apply operation (write JSON + create/move/touch marker files)
3. git add && git commit -m "close canon-a1b2 reason=completed"
4. keep working, accumulate commits
```

**Sync (when ready or back online):**

```
1. git fetch origin beadwork
2. git rebase origin/beadwork
3. clean rebase? →
   - soft replay: re-derive expected state by replaying intents
     against pre-rebase server state
   - compare only the files touched by local commits
   - match? → git push, done
   - mismatch? → use replayed version, amend commit, git push
4. dirty rebase (conflict on any file)? →
   - git rebase --abort
   - collect intents from local-only commit messages
   - git reset --hard origin/beadwork
   - replay each intent against current state
   - git commit && git push
```

The fast path — a clean rebase — handles the vast majority of cases. But even on a clean rebase, the soft replay validates the result before pushing. Rebase is an optimization; the intents are always the source of truth for correctness. Validation only examines files touched by local commits — not the whole repository.

The slow path fires when rebase hits any conflict. Once one conflict appears, more are likely, so the system skips straight to full intent replay. The structured commit messages become a replay log. "Create canon-a1b2 p1 task 'Fix bug'" is replayable. "Close canon-a1b2" is replayable. They're idempotent operations, not diffs — they can be re-applied to any state.

Some intents may fail on replay — closing an issue someone else already closed, for instance. These failures are deterministic and surface as errors. That's a real conflict a human should see.

The system is never in an inconsistent state. Locally, you always have a valid set of files. On sync, the intents are always verified against truth before pushing. There are no conflict markers, no custom merge drivers, no silent corruption.

### What Disappears

| Beads Concept                      | Beadwork Equivalent                           |
| ---------------------------------- | --------------------------------------------- |
| SQLite / Dolt database             | JSON files + directory structure               |
| JSONL export/import                | Not needed — files are the source of truth    |
| Custom merge drivers               | Not needed — rebase or replay, never merge    |
| Conflict resolution                | Rebase, or intent replay from commit messages |
| Daemon process                     | Not needed                                    |
| Lock files                         | Not needed                                    |
| `bd doctor` / `bd rebuild`         | Not needed — no state to drift                |
| Federation / Dolt remotes          | `git push` / `git pull`                       |
| Sync modes (4 options)             | One mode: `bw sync` (rebase or replay)        |
| `beads-sync` branch / data on main | Orphan branch — the only mode                 |
| `bd vc` subcommands                | Just git                                      |
| Database migrations                | Not needed                                    |
| Compaction                         | Not needed                                    |
| Tombstone system                   | Not needed — closed issues still exist        |
| Index file                         | Not needed — the filesystem is the index      |

### What We Keep

The parts of Beads that actually matter for agent ergonomics:

- Short random issue IDs (e.g., `canon-a1b2`) with collision avoidance
- `--json` output on every command for agent consumption
- `bw ready` — list open issues with no open blockers
- Hierarchical issues (epics containing tasks) via `parent/` directory
- Dependency tracking (`blocks/`) with graph queries
- Configurable issue prefixes per project
- `AGENTS.md` with tool usage instructions
- Priority, type, labels, assignee fields

### CLI Surface

```
bw init [--prefix <name>]    Initialize beadwork (orphan branch + worktree in .git/)
bw create <title> [flags]    Create an issue
bw show <id>                 Show issue details
bw list [filters]            List issues (directory listings + JSON reads)
bw update <id> [flags]       Update an issue
bw close <id> [--reason]     Close an issue
bw reopen <id>               Reopen a closed issue
bw ready                     List open issues with no open blockers
bw link <id> blocks <id>     Create dependency links
bw unlink <id> blocks <id>   Remove dependency links
bw label <id> +bug -frontend Add/remove labels
bw graph [<id>]              Show dependency graph (ASCII default, JSON with --json)
bw sync                      Fetch, rebase (or replay), push
```

No `doctor`. No `migrate`. No `dolt`. No `vc`. No `federation`. No `daemon`. No `admin compact`. No `export`. No `import`. No `rebuild`.

### Implementation

**Language:** Go — single binary, cross-platform, fast filesystem and git operations.

**Estimated scope:** ~1,000–2,000 lines of Go for a fully functional v1. The simplicity of the design is the feature.

**Dependencies:** Git (>= 2.5 for worktree support). Nothing else.

### Platform Considerations

**Windows:** No symlinks, no platform-specific code. JSON files, directories, and git worktrees work everywhere.

**Git worktrees:** Supported since Git 2.5 (July 2015). Universally available on any modern system.

### Philosophy

The best infrastructure is the infrastructure you don't build. Git is the most battle-tested distributed version control system in existence. Every feature Beads reimplemented poorly — versioning, branching, sync, collaboration, history, conflict resolution — git already does well.

Beadwork's job is to be a thin CLI that manipulates files. Git's job is everything else.
