package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/repo"
)

// fprintGitContext prints a one-line summary of the current git state.
func fprintGitContext(w Writer, ctx repo.GitContext) {
	location := "Branch"
	if ctx.IsWorktree {
		location = "Worktree"
	}
	state := "clean"
	if ctx.Dirty {
		state = "**uncommitted changes**"
	}
	fmt.Fprintf(w, "%s `%s` · %s · last commit: `%s`\n", location, ctx.Branch, state, ctx.LastCommit)
}
