package main

import (
	"fmt"
	"os"
	"sort"
)

func cmdConfig(args []string) {
	r, _ := mustInitialized()

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: bw config get|set|list\n")
		os.Exit(1)
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			fatal("usage: bw config get <key>")
		}
		val, ok := r.GetConfig(args[1])
		if !ok {
			os.Exit(1)
		}
		fmt.Println(val)

	case "set":
		if len(args) < 3 {
			fatal("usage: bw config set <key> <value>")
		}
		key, value := args[1], args[2]
		if err := r.SetConfig(key, value); err != nil {
			fatal(err.Error())
		}
		intent := fmt.Sprintf("config %s=%s", key, value)
		if err := r.Commit(intent); err != nil {
			fatal("commit failed: " + err.Error())
		}
		fmt.Printf("%s=%s\n", key, value)

	case "list":
		cfg := r.ListConfig()
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s=%s\n", k, cfg[k])
		}

	default:
		fmt.Fprintf(os.Stderr, "usage: bw config get|set|list\n")
		os.Exit(1)
	}
}
