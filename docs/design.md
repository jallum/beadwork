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
archive/
  bw-9z8y.json       (closed issues moved out of active consideration)
```

Every listing query is a directory read. Parent-child relationships use the same marker pattern, with cycle detection preventing circular hierarchies. Two agents working on different issues never touch the same file.

## Attachments

Arbitrary binary or text blobs may be stored alongside an issue under the
`attachments/<ticket-id>/` tree:

```
attachments/
  bw-a1b2/
    design.png
    notes/background.md
```

The path after `<ticket-id>/` is stored verbatim — no normalization, no
basename flattening. Nested paths are allowed. This mirrors the existing
marker-tree convention (e.g. `blocks/<blocker>/<blocked>`).

Reads go through `store.GetAttachment(ticketID, path)`, which walks the
current Beadwork tree and returns the blob bytes; a sentinel
`ErrAttachmentNotFound` is returned when the path is absent. Writes go
through the internal `store.Attach(ticketID, storedPath, content)` helper,
which stages the blob and appends an `attach` intent line (see below).

## Sync

Every CLI operation commits with a structured message that doubles as a replayable intent log:

```
create bw-a1b2 p1 task "Fix auth bug"
close bw-a1b2 reason="completed"
link bw-a1b2 blocks bw-c3d4
delete bw-a1b2
comment bw-a1b2 "Fixed in latest deploy"
attach bw-a1b2 design.png
```

### The `attach` intent

```
attach <ticket-id> <path-verbatim>
```

Tokens are separated by a single space. `<ticket-id>` matches the existing
ticket-id format (e.g. `bw-[a-z0-9]+`). `<path-verbatim>` is the rest of
the line up to newline — it may contain `/` and `.`, must not contain a
trailing whitespace character, and must not embed a newline.

Multiple `attach` lines may appear in a single commit message, after the
primary intent line:

```
review bw-parent: create bw-review, move parent to in_progress
attach bw-review apps/octopus/lib/foo.ex
attach bw-review apps/octopus/lib/bar.ex
```

**Replay semantics.** Given `attach <ticket-id> <path>` in a commit message
being replayed: look up the blob oid at `attachments/<ticket-id>/<path>` in
the pre-replay commit tree (git keeps objects in the object database even
after a ref reset). Re-stage the tree entry at that path with that blob
oid. If the blob is missing from the ODB, the replay fails loudly with an
error — attachments are never silently dropped.

### The `archive` intent

```
archive <ticket-id> [--close] [--detach]
```

`bw archive` moves a closed issue out of the live tree into
`archive/<ticket-id>.json` and removes its `status/`, `labels/`, and
`blocks/` markers. Archived issues leave the live ID space (so `ready`,
`blocked`, `list`, and ID resolution skip them) but remain in git; the
move is one-way. Because the issue is detached from the dependency graph,
its blockers/dependents lose the edge and open children are orphaned —
the same cleanup `delete` performs, except the JSON is preserved under
`archive/` rather than removed.

`--close` closes a still-open issue before archiving it; `--detach`
authorizes severing edges to *open* dependents/children (without it,
archive refuses when live work is attached). A bulk `bw archive --before
<date>` sweep resolves to a concrete set of ticket IDs at commit time and
emits one `archive <ticket-id>` line per issue — never a date expression —
so replay is deterministic. Archived IDs are excluded from ID generation
and explicit-ID validation, so they are never recycled.

**Replay semantics.** `archive <ticket-id>` replays by re-running the
archive against the current tree with the given flags. Multiple `archive`
lines may appear in one commit message (the bulk sweep); each replays
independently and converges to the same end state.

`bw sync` fetches, rebases, and pushes. If rebase conflicts, it replays intents from commit messages against the current remote state. No merge drivers, no lock files, no custom conflict resolution.