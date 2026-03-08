# Markdown-Native Output

Architecture for beadwork's markdown-first rendering pipeline.

## Tokenized Markdown

Render functions in `internal/md/render.go` produce **tokenized markdown** —
markdown with lightweight semantic tokens embedded inline. Two resolution passes
produce the final output:

```
render → tokenized markdown
           ├─ ResolveMarkdown() → clean markdown (agents)
           └─ ResolveTTY(width) → plain text with TTY extras → wrap → ANSI colorize (humans)
```

## Token Vocabulary

| Token | → Markdown | → TTY |
|-------|-----------|-------|
| `{status:open}` | `○` | `○` |
| `{status:closed}` | `✓` | `✓` (green) |
| `{status:in_progress}` | `◐` | `◐` (yellow) |
| `{status:blocked}` | `⊘` | `⊘` (red) |
| `{status:deferred}` | `❄` | `❄` (dim) |
| `{p:0}`..`{p:4}` | `P0`..`P4` | `● P0`..`● P4` (priority color) |
| `{id:bw-xyz}` | `bw-xyz` | `bw-xyz` (cyan) |
| `{type:epic}` | `[EPIC]` | `[EPIC]` |
| `{check:done}` | `[x]` | `✓` |
| `{check:open}` | `[ ]` | `[ ]` |
| `{dep:blocks:id}` | `[blocks: id]` | dim `[blocks:` red `id` dim `]` |
| `{dep:blocked_by:id}` | `[blocked by: id]` | same |

## Escaping

User-provided text (descriptions, comments, titles) is escaped before
interpolation via `md.Escape(s)`, which replaces `{` with `\x00{` to prevent
token collisions. Resolution passes reverse this.

## TTY Pipeline

The TTY path has three stages:

1. **Resolve tokens → plain TTY text.** `{p:1}` → `● P1`, `{id:bw-xyz}` → `bw-xyz`.
   Color markers (`\x01kind\x02...\x01end\x02`) are inserted but no ANSI yet.
2. **Wrap.** `wrap.Text(plainTTY, width)` operates on correctly-sized plain text.
3. **Colorize.** Apply ANSI to the wrapped output by resolving color markers and
   styling markdown syntax (#headings, **bold**, `code`, ~~strike~~).

## emit() Pattern

The `emit(w Writer, tokenized string)` function in `cmd/bw/emit.go` routes
tokenized markdown through the appropriate resolver:

- `w.IsRaw()` → dump tokenized text as-is (for integration testing)
- `w.IsTTY()` → `md.ResolveTTY(tokenized, w.Width())`
- else → `md.ResolveMarkdown(tokenized)`

## --x-raw Flag

Undocumented global flag that selects a RawWriter, passing tokenized text
through without resolution. Useful for integration tests (assert on `{p:1}`
and `{id:bw-xyz}` tokens directly) and debugging. The `--x-` prefix marks
it as internal (consistent with `--x-render-as`).

## TTY-Only Affordances

The TTY path adds things that don't exist in markdown:
- `●` priority dot with color (from `{p:N}`)
- Ready footer (separator, count, legend)
- Text wrapping to terminal width
- ANSI coloring of markdown headings, bold, code, strikethrough
