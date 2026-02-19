package main

import (
	"fmt"

	"github.com/j5n/beadwork/prompts"
)

func cmdOnboard() {
	fmt.Printf(prompts.Onboard, prompts.AgentsMD)
}
