package main

import "fmt"

func cmdInit(args []string) {
	prefix := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--prefix" && i+1 < len(args) {
			prefix = args[i+1]
			i++
		}
	}

	r := mustRepo()
	if r.IsInitialized() {
		fatal("beadwork already initialized")
	}
	if err := r.Init(prefix); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("initialized beadwork (prefix: %s)\n", r.Prefix)
}
