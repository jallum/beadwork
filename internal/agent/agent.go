// Package agent detects which AI coding agent (if any) is invoking the current process.
//
// Detection relies on environment variables that agents set in subprocesses.
// Each entry is marked with a confidence level:
//
//   - confirmed: verified firsthand or documented by the agent's maintainers
//   - reported:  community-reported but not verified; may be absent or buggy
//   - unknown:   no known signal; detection not yet possible
package agent

import "os"

// Agent identifies an AI coding agent.
type Agent struct {
	// Name is a short, stable identifier (e.g. "claude-code", "gemini-cli").
	Name string

	// EnvVar is the environment variable used for detection, if any.
	EnvVar string

	// Confidence describes how reliable the detection signal is.
	// One of: "confirmed", "reported".
	Confidence string
}

// probe is a detection check: an env var to look for and the Agent it identifies.
type probe struct {
	envVar string
	agent  Agent
}

// probes are checked in order; first match wins.
var probes = []probe{
	{
		envVar: "CLAUDECODE",
		agent: Agent{
			Name:       "claude-code",
			EnvVar:     "CLAUDECODE",
			Confidence: "confirmed",
		},
	},
	{
		envVar: "GEMINI_CLI",
		agent: Agent{
			Name:       "gemini-cli",
			EnvVar:     "GEMINI_CLI",
			Confidence: "confirmed",
		},
	},
	{
		envVar: "CURSOR_AGENT",
		agent: Agent{
			Name:       "cursor",
			EnvVar:     "CURSOR_AGENT",
			Confidence: "reported",
		},
	},
}

// Detect returns the agent invoking the current process, or nil if none is detected.
func Detect() *Agent {
	return DetectFrom(os.LookupEnv)
}

// DetectFrom is like Detect but uses a custom lookup function, useful for testing.
func DetectFrom(lookupEnv func(string) (string, bool)) *Agent {
	for _, p := range probes {
		if v, ok := lookupEnv(p.envVar); ok && v != "" {
			a := p.agent
			return &a
		}
	}
	return nil
}
