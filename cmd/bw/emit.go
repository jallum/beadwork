package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/md"
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
