package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/prompts"
)

func cmdOnboard(w io.Writer) error {
	fmt.Fprintf(w, prompts.Onboard, prompts.AgentsMD)
	return nil
}
