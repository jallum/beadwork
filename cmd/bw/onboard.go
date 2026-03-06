package main

import (
	"fmt"

	"github.com/jallum/beadwork/prompts"
)

func cmdOnboard(w Writer) error {
	fmt.Fprintf(w, prompts.Onboard, prompts.AgentsMD)
	return nil
}
