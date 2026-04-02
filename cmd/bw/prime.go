package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/tmpl"
	"github.com/jallum/beadwork/prompts"
)

type PrimeData struct {
	Prefix            string
	WorktreeDirty     bool
	Git               repo.GitContext
	OverdueCount      int
	ExpiredDeferrals  string
}

func cmdPrime(store *issue.Store, _ []string, w Writer) error {
	r := store.Committer.(*repo.Repo)
	cfg := r.ListConfig()
	gitCtx := r.GetGitContext()

	overdueIssues, _ := store.List(issue.Filter{Overdue: true})

	// Find expired deferrals for the reminders section.
	now := store.Now()
	deferredIssues, _ := store.List(issue.Filter{Status: "deferred"})
	var expiredLines []string
	for _, iss := range deferredIssues {
		if issue.IsDeferralExpired(iss.DeferUntil, now) {
			expiredLines = append(expiredLines, md.IssueOneLinerWithDue(iss, now, nil))
		}
	}

	data := PrimeData{
		Prefix:           cfg["prefix"],
		WorktreeDirty:    gitCtx.Dirty,
		Git:              gitCtx,
		OverdueCount:     len(overdueIssues),
		ExpiredDeferrals: strings.Join(expiredLines, "\n"),
	}

	bwFn := func(args ...string) string {
		if cmd := commandMap[args[0]]; cmd != nil {
			var buf bytes.Buffer
			cmd.Run(store, args[1:], TokenWriter(&buf))
			return strings.TrimRight(buf.String(), "\n")
		}
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, "prime", prompts.Prime, data, bwFn); err != nil {
		return err
	}

	out := strings.Trim(buf.String(), "\n")
	fmt.Fprint(w, out)
	fmt.Fprintln(w)
	return nil
}
