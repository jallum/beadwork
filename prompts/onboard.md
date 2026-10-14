# Onboarding

Add this to your agent instructions file (CLAUDE.md, GEMINI.md, COPILOT.md, etc.):

```text
{{ .Snippet }}
```

## How it works

- `bw prime` loads workflow context at session start — once it runs, the agent has everything it needs
- The snippet only needs this minimal pointer, not full instructions

## Customize workflow guidance

`bw config list` shows `prompts.prime` and `prompts.start`. Empty values use the
built-in prompts. Select custom Markdown files:

```sh
bw config set prompts.prime .beadwork/prompts/prime.md
bw config set prompts.start .beadwork/prompts/start.md
```

Relative paths use the current checkout's repository root. Set either value to
`""` to restore its default.
