bw Onboarding

Add this minimal snippet to AGENTS.md (or create it):

--- BEGIN AGENTS.MD CONTENT ---
## Issue Tracking

This project uses **bw (beadwork)** for issue tracking.
Run `bw prime` for workflow context.

**Quick reference:**
- `bw ready` - Find unblocked work
- `bw create "Title" --type task --priority 2` - Create issue
- `bw close <id>` - Complete work
- `bw sync` - Sync with git (run at session end)

For full workflow details: `bw prime`
--- END AGENTS.MD CONTENT ---

How it works:
  - bw prime provides dynamic workflow context
  - AGENTS.md only needs this minimal pointer, not full instructions
