package main

import (
	"fmt"

	"github.com/jallum/beadwork/prompts"
)

func cmdOnboard() {
	fmt.Printf(prompts.Onboard, prompts.AgentsMD)
}
