package recap

import (
	"regexp"
	"strings"
	"time"
)

// Intent patterns — anchored to the start of a line.
var (
	createRe    = regexp.MustCompile(`^create\s+(\S+)`)
	closeRe     = regexp.MustCompile(`^close\s+(\S+)`)
	startRe     = regexp.MustCompile(`^start\s+(\S+)`)
	updateRe    = regexp.MustCompile(`^update\s+(\S+)`)
	reopenRe    = regexp.MustCompile(`^reopen\s+(\S+)`)
	deferRe     = regexp.MustCompile(`^defer\s+(\S+)`)
	undeferRe   = regexp.MustCompile(`^undefer\s+(\S+)`)
	commentRe   = regexp.MustCompile(`^comment\s+(\S+)`)
	linkRe      = regexp.MustCompile(`^link\s+(\S+)\s+blocks\s+(\S+)`)
	unlinkRe    = regexp.MustCompile(`^unlink\s+(\S+)\s+blocks\s+(\S+)`)
	deleteRe    = regexp.MustCompile(`^delete\s+(\S+)`)
	labelRe     = regexp.MustCompile(`^label\s+(\S+)`)
	unblockedRe = regexp.MustCompile(`^unblocked\s+(\S+)$`)
)

// ParseIntent extracts events from a beadwork commit message.
// The first line is the primary intent; subsequent lines may contain
// secondary events (e.g., "unblocked <id>").
func ParseIntent(message string, ts time.Time) []Event {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) == 0 {
		return nil
	}

	var events []Event
	first := strings.TrimSpace(lines[0])

	switch {
	case createRe.MatchString(first):
		m := createRe.FindStringSubmatch(first)
		detail := strings.TrimSpace(first[len(m[0]):])
		events = append(events, Event{Type: "create", ID: m[1], Time: ts, Detail: detail})
	case closeRe.MatchString(first):
		m := closeRe.FindStringSubmatch(first)
		detail := strings.TrimSpace(first[len(m[0]):])
		events = append(events, Event{Type: "close", ID: m[1], Time: ts, Detail: detail})
	case startRe.MatchString(first):
		m := startRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "start", ID: m[1], Time: ts})
	case updateRe.MatchString(first):
		m := updateRe.FindStringSubmatch(first)
		detail := strings.TrimSpace(first[len(m[0]):])
		events = append(events, Event{Type: "update", ID: m[1], Time: ts, Detail: detail})
	case reopenRe.MatchString(first):
		m := reopenRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "reopen", ID: m[1], Time: ts})
	case deferRe.MatchString(first):
		m := deferRe.FindStringSubmatch(first)
		detail := strings.TrimSpace(first[len(m[0]):])
		events = append(events, Event{Type: "defer", ID: m[1], Time: ts, Detail: detail})
	case undeferRe.MatchString(first):
		m := undeferRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "undefer", ID: m[1], Time: ts})
	case commentRe.MatchString(first):
		m := commentRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "comment", ID: m[1], Time: ts})
	case linkRe.MatchString(first):
		m := linkRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "link", ID: m[1], Time: ts, Detail: "blocks " + m[2]})
	case unlinkRe.MatchString(first):
		m := unlinkRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "unlink", ID: m[1], Time: ts, Detail: "blocks " + m[2]})
	case deleteRe.MatchString(first):
		m := deleteRe.FindStringSubmatch(first)
		events = append(events, Event{Type: "delete", ID: m[1], Time: ts})
	case labelRe.MatchString(first):
		m := labelRe.FindStringSubmatch(first)
		detail := strings.TrimSpace(first[len(m[0]):])
		events = append(events, Event{Type: "label", ID: m[1], Time: ts, Detail: detail})
	}

	// Parse secondary events from lines >= 2 (e.g., "unblocked <id>").
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if m := unblockedRe.FindStringSubmatch(line); m != nil {
			events = append(events, Event{Type: "unblocked", ID: m[1], Time: ts})
		}
	}

	return events
}
