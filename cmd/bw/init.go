package main

import "fmt"

func cmdInit(args []string) {
	prefix := ""
	force := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--prefix":
			if i+1 < len(args) {
				prefix = args[i+1]
				i++
			}
		case "--force":
			force = true
		}
	}

	r := mustRepo()
	if force {
		if err := r.ForceReinit(prefix); err != nil {
			fatal(err.Error())
		}
		fmt.Printf("reinitialized beadwork (prefix: %s)\n", r.Prefix)
		return
	}
	if r.IsInitialized() {
		fatal("beadwork already initialized")
	}
	if err := r.Init(prefix); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("initialized beadwork (prefix: %s)\n", r.Prefix)
}
