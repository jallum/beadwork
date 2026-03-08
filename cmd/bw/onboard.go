package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/tmpl"
	"github.com/jallum/beadwork/prompts"
)

type OnboardData struct {
	Snippet string
}

func cmdOnboard(w Writer) error {
	data := OnboardData{
		Snippet: prompts.AgentsMD,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, "onboard", prompts.Onboard, data, nil); err != nil {
		return err
	}

	out := strings.Trim(buf.String(), "\n")
	fmt.Fprint(w, out)
	fmt.Fprintln(w)
	return nil
}
