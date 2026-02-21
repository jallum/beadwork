package main

import (
	"fmt"
	"time"

	"github.com/jallum/beadwork/internal/issue"
)

type DeferArgs struct {
	ID   string
	Date string
	JSON bool
}

func parseDeferArgs(raw []string) (DeferArgs, error) {
	if len(raw) < 2 {
		return DeferArgs{}, fmt.Errorf("usage: bw defer <id> <date>")
	}
	a, err := ParseArgs(raw[2:], nil, []string{"--json"})
	if err != nil {
		return DeferArgs{}, err
	}
	return DeferArgs{ID: raw[0], Date: raw[1], JSON: a.JSON()}, nil
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return nil
}

func cmdDefer(store *issue.Store, args []string, w Writer) error {
	da, err := parseDeferArgs(args)
	if err != nil {
		return err
	}

	if err := validateDate(da.Date); err != nil {
		return err
	}

	status := "deferred"
	iss, err := store.Update(da.ID, issue.UpdateOpts{
		Status:     &status,
		DeferUntil: &da.Date,
	})
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("defer %s until %s", iss.ID, da.Date)
	if err := store.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if da.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "deferred %s until %s\n", iss.ID, da.Date)
	}
	return nil
}
