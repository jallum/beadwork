package prompts

import _ "embed"

//go:generate cp ../AGENTS.md agents.md

//go:embed agents.md
var AgentsMD string

//go:embed onboard.md
var Onboard string

//go:embed prime.md
var Prime string
