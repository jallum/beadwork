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

	sections := map[string]func(io.Writer){
		"STATE": func(_ io.Writer) {
			flush()
			cmdReady(store, nil, w)
		},
	}

	template.Process(&buf, prompts.Prime, r.ListConfig(), sections)
	flush()

	return nil
}
