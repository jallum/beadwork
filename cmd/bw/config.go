package main

import (
	"fmt"
	"sort"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type ConfigArgs struct {
	Subcmd string // "get", "set", "list"
	Key    string // for get/set
	Value  string // for set
}

func parseConfigArgs(raw []string) (ConfigArgs, error) {
	if len(raw) == 0 {
		return ConfigArgs{}, fmt.Errorf("usage: bw config get|set|list")
	}
	ca := ConfigArgs{Subcmd: raw[0]}
	switch ca.Subcmd {
	case "get":
		if len(raw) < 2 {
			return ca, fmt.Errorf("usage: bw config get <key>")
		}
		ca.Key = raw[1]
	case "set":
		if len(raw) < 3 {
			return ca, fmt.Errorf("usage: bw config set <key> <value>")
		}
		ca.Key = raw[1]
		ca.Value = raw[2]
	case "list":
		// no additional args
	default:
		return ca, fmt.Errorf("usage: bw config get|set|list")
	}
	return ca, nil
}

func cmdConfig(store *issue.Store, args []string, w Writer, cfg *config.Config) (*config.Config, error) {
	ca, err := parseConfigArgs(args)
	if err != nil {
		return nil, err
	}
	r := store.Committer.(*repo.Repo)

	switch ca.Subcmd {
	case "get":
		val, ok := r.GetConfig(ca.Key)
		if !ok {
			return nil, fmt.Errorf("key not found: %s", ca.Key)
		}
		fmt.Fprintln(w, val)

	case "set":
		// Prefix changes are special: they atomically rename the active
		// (open + in_progress) issues so cross-repo lookups stay consistent.
		// Closed issues keep their old prefix as a historical record.
		if ca.Key == "prefix" {
			return nil, renamePrefix(store, r, ca.Value, w, cfg)
		}
		if err := r.SetConfig(ca.Key, ca.Value); err != nil {
			return nil, err
		}
		intent := fmt.Sprintf("config %s=%s", ca.Key, ca.Value)
		if err := r.Commit(intent); err != nil {
			return nil, fmt.Errorf("commit failed: %w", err)
		}
		fmt.Fprintf(w, "%s=%s\n", ca.Key, ca.Value)

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
	}
	return nil, nil
}
