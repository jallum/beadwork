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