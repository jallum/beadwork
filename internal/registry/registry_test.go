package registry

import (
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/config"
)

func emptyCfg(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "cfg.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestPathsEmpty(t *testing.T) {
	cfg := emptyCfg(t)
	if got := Paths(cfg); len(got) != 0 {
		t.Errorf("Paths = %v, want empty", got)
	}
}

func TestRegisterAndPaths(t *testing.T) {
	cfg := emptyCfg(t)

	cfg2 := Register(cfg, "/a")
	if cfg2 == cfg {
		t.Fatal("Register returned same pointer for new path")
	}

	paths := Paths(cfg2)
	if len(paths) != 1 || paths[0] != "/a" {
		t.Errorf("Paths = %v, want [/a]", paths)
	}

	// Original unchanged.
	if len(Paths(cfg)) != 0 {
		t.Error("Register mutated original config")
	}
}

func TestRegisterIdempotent(t *testing.T) {
	cfg := emptyCfg(t)
	cfg = Register(cfg, "/a")
	cfg2 := Register(cfg, "/a")
	if cfg2 != cfg {
		t.Error("duplicate Register returned new pointer")
	}
}

func TestResolveNoRepos(t *testing.T) {
	cfg := emptyCfg(t)
	if _, ok := Resolve(cfg, "test"); ok {
		t.Error("Resolve found a match in empty registry")
	}
}
