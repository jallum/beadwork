package agent

import "testing"

func TestDetectFrom(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		want   string // expected Agent.Name, or "" for nil
		wantOK bool
	}{
		{
			name:   "no agent",
			env:    map[string]string{},
			want:   "",
			wantOK: false,
		},
		{
			name:   "claude code",
			env:    map[string]string{"CLAUDECODE": "1"},
			want:   "claude-code",
			wantOK: true,
		},
		{
			name:   "gemini cli",
			env:    map[string]string{"GEMINI_CLI": "1"},
			want:   "gemini-cli",
			wantOK: true,
		},
		{
			name:   "cursor",
			env:    map[string]string{"CURSOR_AGENT": "1"},
			want:   "cursor",
			wantOK: true,
		},
		{
			name:   "first match wins",
			env:    map[string]string{"CLAUDECODE": "1", "GEMINI_CLI": "1"},
			want:   "claude-code",
			wantOK: true,
		},
		{
			name:   "empty value ignored",
			env:    map[string]string{"CLAUDECODE": ""},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				v, ok := tt.env[key]
				return v, ok
			}
			got := DetectFrom(lookup)
			if !tt.wantOK {
				if got != nil {
					t.Fatalf("want nil, got %q", got.Name)
				}
				return
			}
			if got == nil {
				t.Fatal("want non-nil, got nil")
			}
			if got.Name != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got.Name)
			}
		})
	}
}
