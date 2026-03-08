package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/md"
	"github.com/jallum/beadwork/internal/repo"
)

// emit resolves tokenized markdown according to the writer's mode and prints it.
func emit(w Writer, tokenized string) {
	if w.IsRaw() {
		fmt.Fprint(w, tokenized)
	} else if w.IsTTY() {
		fmt.Fprint(w, md.ResolveTTY(tokenized, w.Width()))
	} else {
		fmt.Fprint(w, md.ResolveMarkdown(tokenized))
	}
}

// emitln is like emit but appends a newline.
func emitln(w Writer, tokenized string) {
	emit(w, tokenized)
	fmt.Fprintln(w)
}

// emitGitContext prints a one-line summary of the current git state.
func emitGitContext(w Writer, ctx repo.GitContext) {
	location := "Branch"
	if ctx.IsWorktree {
		location = "Worktree"
	}
	state := "clean"
	if ctx.Dirty {
		state = "**uncommitted changes**"
	}
	emitln(w, fmt.Sprintf("%s `%s` · %s · last commit: `%s`", location, ctx.Branch, state, ctx.LastCommit))
}
