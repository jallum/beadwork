package registry

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveFrom(t *testing.T) {
	homeDir := "/home/testuser"
	homeFn := func() (string, error) { return homeDir, nil }
	errHome := func() (string, error) { return "", errors.New("no home") }

	tests := []struct {
		name    string
		envHome string
		homeFn  func() (string, error)
		want    string
	}{
		{
			name:    "BEADWORK_HOME takes precedence",
			envHome: "/custom/beadwork",
			want:    "/custom/beadwork",
		},
		{
			name:   "falls back to ~/.beadwork",
			homeFn: homeFn,
			want:   filepath.Join(homeDir, ".beadwork"),
		},
		{
			name:   "home dir error uses relative path",
			homeFn: errHome,
			want:   ".beadwork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.homeFn
			if fn == nil {
				fn = homeFn
			}
			got := resolveFrom(tt.envHome, fn)
			if got != tt.want {
				t.Errorf("resolveFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}
