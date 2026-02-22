package main

import (
	"bytes"
	"fmt"

	"github.com/jallum/beadwork/internal/template"
	"github.com/jallum/beadwork/prompts"
)

func cmdOnboard(w Writer) error {
	text := fmt.Sprintf(prompts.Onboard, prompts.AgentsMD)
	var buf bytes.Buffer
	template.Process(&buf, text, nil, nil) // no conditionals in onboard
	fmt.Fprint(w, buf.String())
	return nil
}
