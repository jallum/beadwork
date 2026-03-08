# Design

## Storage

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

## Sync

Every CLI operation commits with a structured message that doubles as a replayable intent log:

```
create bw-a1b2 p1 task "Fix auth bug"
close bw-a1b2 reason="completed"
link bw-a1b2 blocks bw-c3d4
delete bw-a1b2
comment bw-a1b2 "Fixed in latest deploy"
```

`bw sync` fetches, rebases, and pushes. If rebase conflicts, it replays intents from commit messages against the current remote state. No merge drivers, no lock files, no custom conflict resolution.

## Migrating Between Beads and Beadwork

Both tools use a shared JSONL interchange format, so migrating is an export/import round-trip. Issue IDs, dependencies, and parent-child relationships are preserved.

### Beads to Beadwork

```bash
bw init                           # initialize beadwork in the same repo
bd export | bw import -           # pipe directly from beads
bw sync                           # push the beadwork branch
```

### Beadwork to Beads

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
