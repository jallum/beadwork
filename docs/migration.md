# Migrating Between Beads and Beadwork

Both tools use a shared JSONL interchange format, so migrating is an export/import round-trip. Issue IDs, dependencies, and parent-child relationships are preserved.

## Beads to Beadwork

```bash
bw init                           # initialize beadwork in the same repo
bd export | bw import -           # pipe directly from beads
bw sync                           # push the beadwork branch
```

## Beadwork to Beads

```bash
bw export > issues.jsonl          # export from beadwork
bd import -i issues.jsonl         # import into beads
```

Use `--status open` on export to migrate only open issues, or `--dry-run` on import to preview before committing. Duplicate IDs are skipped automatically.

## Field Mapping

| Beads | Beadwork | Notes |
|-------|----------|-------|
| `owner` | `assignee` | |
| `issue_type` | `type` | |
| `created_at` | `created` | |
| `dependencies` | `blocks` / `blocked_by` / `parent` | Flattened into separate fields |
| `comments` | `comments` | Text, author, timestamp preserved |
| — | `labels` | Beadwork-only; not present in beads |
