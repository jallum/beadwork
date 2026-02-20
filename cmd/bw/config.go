package main

import (
	"fmt"
	"io"
	"sort"
)

func cmdConfig(args []string, w io.Writer) error {
	r, _, err := getInitialized()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: bw config get|set|list")
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: bw config get <key>")
		}
		val, ok := r.GetConfig(args[1])
		if !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
		fmt.Fprintln(w, val)

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: bw config set <key> <value>")
		}
		key, value := args[1], args[2]
		if err := r.SetConfig(key, value); err != nil {
			return err
		}
		intent := fmt.Sprintf("config %s=%s", key, value)
		if err := r.Commit(intent); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}
		fmt.Fprintf(w, "%s=%s\n", key, value)

	case "list":
		cfg := r.ListConfig()
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s=%s\n", k, cfg[k])
		}

	default:
		return fmt.Errorf("usage: bw config get|set|list")
	}
	return nil
}
