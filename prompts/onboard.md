Beadwork (bw) Onboarding

Add this snippet to your agent instructions file (CLAUDE.md, GEMINI.md, COPILOT.md, etc.):

--- BEGIN SNIPPET ---
%s
--- END SNIPPET ---

How it works:

- `bw prime` loads workflow context at session start — once it runs, the agent has everything it needs
- The snippet only needs this minimal pointer, not full instructions
