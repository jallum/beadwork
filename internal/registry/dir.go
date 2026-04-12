package registry

import (
	"os"
	"path/filepath"
)

const dirName = ".beadwork"

// DefaultDir returns the beadwork home directory. BEADWORK_HOME overrides
// it; otherwise it falls back to ~/.beadwork. os.UserHomeDir honors $HOME,
// so callers who want a different home can point HOME at it.
func DefaultDir() string {
	return resolveFrom(os.Getenv("BEADWORK_HOME"), os.UserHomeDir)
}

func resolveFrom(envHome string, homeFn func() (string, error)) string {
	if envHome != "" {
		return envHome
	}
	home, err := homeFn()
	if err != nil {
		return dirName
	}
	return filepath.Join(home, dirName)
}
