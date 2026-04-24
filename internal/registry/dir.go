package registry

import (
	"os"
	"path/filepath"
)

const defaultFileName = ".bw"

// DefaultPath returns the path to the registry file.
// BW_REGISTRY overrides it; otherwise it falls back to ~/.bw.
func DefaultPath() string {
	if v := os.Getenv("BW_REGISTRY"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultFileName
	}
	return filepath.Join(home, defaultFileName)
}
