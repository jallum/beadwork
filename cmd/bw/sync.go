package main

import (
	"fmt"
	"os"

	"github.com/jallum/beadwork/internal/intent"
)

func cmdSync(args []string) {
	r, store := mustInitialized()
	_ = args

	status, intents, err := r.Sync()
	if err != nil {
		fatal(err.Error())
	}

	if status == "needs replay" {
		fmt.Printf("rebase conflict — replaying %d intent(s)...\n", len(intents))
		errs := intent.Replay(r, store, intents)
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  warning: %s\n", e)
			}
		}
		if err := r.Push(); err != nil {
			fatal("push after replay failed: " + err.Error())
		}
		fmt.Println("replayed and pushed")
	} else {
		fmt.Println(status)
	}
}
