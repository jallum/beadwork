// Package config provides a lightweight global configuration backed by a
// YAML file (~/.bw by default, overridden by $BW_CONFIG). The config is a
// nested map[string]any — feature packages provide their own typed accessors.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultFileName = ".bw"

// Config holds the in-memory configuration and the path it was loaded from.
type Config struct {
	data map[string]any
	path string
}

// Load reads the config from path. If the file does not exist, returns an
// empty config that will save to path.
func Load(path string) (*Config, error) {
	c := &Config{
		data: make(map[string]any),
		path: path,
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(raw, &c.data); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if c.data == nil {
		c.data = make(map[string]any)
	}
	return c, nil
}

// Save atomically writes the config to the path it was loaded from.
func (c *Config) Save() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	raw, err := yaml.Marshal(c.data)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// Path returns the file path this config was loaded from.
func (c *Config) Path() string { return c.path }

// Data returns the raw underlying map.
func (c *Config) Data() map[string]any { return c.data }

// DefaultPath returns the config file path.
// BW_CONFIG overrides it; otherwise falls back to ~/.bw.
func DefaultPath() string {
	if v := os.Getenv("BW_CONFIG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultFileName
	}
	return filepath.Join(home, defaultFileName)
}

// Get retrieves a value by dot-separated key path (e.g. "registry.repos").
// Returns nil if any segment is missing.
func (c *Config) Get(key string) any {
	parts := strings.Split(key, ".")
	var cur any = c.data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

// Set returns a new Config with value written at the dot-separated key path.
// If the value is unchanged, returns the receiver (same pointer). Only maps
// along the modified path are copied; sibling subtrees are shared.
func (c *Config) Set(key string, value any) *Config {
	parts := strings.Split(key, ".")
	newData := setPath(c.data, parts, value)
	if newData == nil {
		return c
	}
	return &Config{data: newData, path: c.path}
}

// setPath recursively walks parts, returning a new map with the value set at
// the leaf. Returns nil if nothing changed.
func setPath(m map[string]any, parts []string, value any) map[string]any {
	key := parts[0]
	if len(parts) == 1 {
		if reflect.DeepEqual(m[key], value) {
			return nil
		}
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		out[key] = value
		return out
	}
	child, _ := m[key].(map[string]any)
	if child == nil {
		child = make(map[string]any)
	}
	newChild := setPath(child, parts[1:], value)
	if newChild == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	out[key] = newChild
	return out
}

// String returns the string at key, or "" if missing or not a string.
func (c *Config) String(key string) string {
	v, _ := c.Get(key).(string)
	return v
}

// Bool returns the bool at key, or false if missing or not a bool.
func (c *Config) Bool(key string) bool {
	v, _ := c.Get(key).(bool)
	return v
}

// StringSlice returns the string slice at key. YAML sequences of strings
// unmarshal as []any, so this handles the conversion.
func (c *Config) StringSlice(key string) []string {
	raw := c.Get(key)
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// Section returns the sub-map at key, or nil if missing.
func (c *Config) Section(key string) map[string]any {
	v, _ := c.Get(key).(map[string]any)
	return v
}
