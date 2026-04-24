package registry

import (
	"os"
	"path/filepath"
)

const defaultFileName = ".bw"

// DefaultPath returns the path to the registry file.
// BEADWORK_HOME overrides it; otherwise it falls back to ~/.bw.
func DefaultPath() string {
	if v := os.Getenv("BEADWORK_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultFileName
	}
	return filepath.Join(home, defaultFileName)
}
