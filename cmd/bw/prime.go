package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/template"
	"github.com/jallum/beadwork/prompts"
)

func cmdPrime(store *issue.Store, _ []string, w Writer) error {
	r := store.Committer.(*repo.Repo)

	var buf bytes.Buffer
	firstFlush := true

	flush := func() {
		s := strings.Trim(buf.String(), "\n")
		buf.Reset()
		if s == "" {
			return
		}
		if !firstFlush {
			fmt.Fprintln(w)
		}
		fmt.Fprint(w, styleMD(w, s))
		fmt.Fprintln(w)
		firstFlush = false
	}

	cmdFn := func(args []string, _ io.Writer) {
		if cmd := commandMap[args[0]]; cmd != nil {
			flush()
			cmd.Run(store, args[1:], w)
		}
	}

	template.ProcessWithCommands(&buf, prompts.Prime, r.ListConfig(), nil, cmdFn)
	flush()

	return nil
}
