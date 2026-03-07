package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/tmpl"
	"github.com/jallum/beadwork/prompts"
)

type PrimeData struct {
	Prefix         string
	WorktreeDirty  bool
}

func cmdPrime(store *issue.Store, _ []string, w Writer) error {
	r := store.Committer.(*repo.Repo)
	cfg := r.ListConfig()

	data := PrimeData{
		Prefix:        cfg["prefix"],
		WorktreeDirty: r.WorktreeDirty(),
	}

	bwFn := func(args ...string) string {
		if cmd := commandMap[args[0]]; cmd != nil {
			var buf bytes.Buffer
			cmd.Run(store, args[1:], PlainWriter(&buf))
			return strings.TrimRight(buf.String(), "\n")
		}
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, "prime", prompts.Prime, data, bwFn); err != nil {
		return err
	}

	out := strings.Trim(buf.String(), "\n")
	emit(w, out)
	fmt.Fprintln(w)
	return nil
}
