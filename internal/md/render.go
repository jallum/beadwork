// Package md produces tokenized markdown output for beadwork commands.
// Tokenized markdown embeds lightweight semantic tokens ({status:open},
// {id:bw-xyz}, etc.) that are resolved to clean markdown or ANSI-colored
// TTY output by ResolveMarkdown and ResolveTTY respectively.
package md

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

// escChar is the escape prefix used to protect literal { in user content.
const escChar = "\x00"

// Escape replaces { with an escape sequence in user-provided text to
// prevent token collisions during resolution.
func Escape(s string) string {
	return strings.ReplaceAll(s, "{", escChar+"{")
}

// IssueSummary returns a # heading line with status, id, optional type tag,
// and title, followed by optional Parent: and Labels: lines.
func IssueSummary(iss *issue.Issue) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(statusToken(iss.Status))
	b.WriteByte(' ')
	b.WriteString(idToken(iss.ID))

	if iss.Type != "" && iss.Type != "task" {
		b.WriteByte(' ')
		b.WriteString(typeToken(iss.Type))
	}

	b.WriteString(" \u2014 ")
	b.WriteString(Escape(iss.Title))

	if iss.DeferUntil != "" {
		b.WriteString("\nDeferred: ")
		b.WriteString(iss.DeferUntil)
	}
	if iss.Parent != "" {
		b.WriteString("\nParent: ")
		b.WriteString(iss.Parent)
	}
	if len(iss.Labels) > 0 {
		b.WriteString("\nLabels: ")
		b.WriteString(strings.Join(iss.Labels, ", "))
	}

	return b.String()
}

// IssueOneLiner returns a compact single-line representation with status,
// id, priority, optional type tag, title, and inline deps.
func IssueOneLiner(iss *issue.Issue) string {
	return issueOneLiner(iss, statusToken(iss.Status), nil)
}

// IssueOneLinerBlocked is like IssueOneLiner but uses the blocked status icon
// when openBlockers is non-empty.
func IssueOneLinerBlocked(iss *issue.Issue, openBlockers []string) string {
	st := statusToken(iss.Status)
	if len(openBlockers) > 0 && iss.Status == "open" {
		st = statusToken("blocked")
	}
	return issueOneLiner(iss, st, nil)
}

// IssueOneLinerFiltered is like IssueOneLiner but filters out closed blockers
// from the dep annotation.
func IssueOneLinerFiltered(iss *issue.Issue, closedBlockers map[string]bool) string {
	return issueOneLiner(iss, statusToken(iss.Status), closedBlockers)
}

func issueOneLiner(iss *issue.Issue, statusTok string, closedBlockers map[string]bool) string {
	var b strings.Builder
	b.WriteString(statusTok)
	b.WriteByte(' ')
	b.WriteString(idToken(iss.ID))
	b.WriteByte(' ')
	b.WriteString(priorityToken(iss.Priority))

	if iss.Type != "" && iss.Type != "task" {
		b.WriteByte(' ')
		b.WriteString(typeToken(iss.Type))
	}

	b.WriteByte(' ')
	b.WriteString(Escape(iss.Title))

	var deps string
	if closedBlockers != nil {
		deps = FormatDepsFiltered(iss, closedBlockers)
	} else {
		deps = FormatDeps(iss)
	}
	if deps != "" {
		b.WriteString(deps)
	}

	return b.String()
}

// Description returns escaped user-provided description text.
func Description(desc string) string {
	return Escape(desc)
}

// Children returns a ## CHILDREN section with checkbox lines.
// Returns "" if children is empty.
func Children(children []*issue.Issue) string {
	if len(children) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## CHILDREN\n")
	for _, child := range children {
		b.WriteString("\n- ")
		if child.Status == "closed" {
			b.WriteString("{check:done}")
		} else {
			b.WriteString("{check:open}")
		}
		b.WriteByte(' ')
		b.WriteString(idToken(child.ID))
		b.WriteByte(' ')
		b.WriteString(priorityToken(child.Priority))
		b.WriteByte(' ')
		b.WriteString(Escape(child.Title))
		b.WriteString(FormatDeps(child))
	}
	return b.String()
}

// BlockedBy returns a ## BLOCKED BY section listing actionable blockers.
// Returns "" if blockers is empty.
func BlockedBy(blockers []*issue.Issue) string {
	if len(blockers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## BLOCKED BY\n")
	for _, bl := range blockers {
		b.WriteString("\n- ")
		b.WriteString(statusToken(bl.Status))
		b.WriteByte(' ')
		b.WriteString(idToken(bl.ID))
		b.WriteByte(' ')
		b.WriteString(priorityToken(bl.Priority))
		b.WriteByte(' ')
		b.WriteString(Escape(bl.Title))
	}
	return b.String()
}

// Unblocks returns a ## UNBLOCKS section listing dependents.
// Returns "" if deps is empty.
func Unblocks(deps []*issue.Issue) string {
	if len(deps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## UNBLOCKS\n")
	for _, dep := range deps {
		b.WriteString("\n- ")
		b.WriteString(statusToken(dep.Status))
		b.WriteByte(' ')
		b.WriteString(idToken(dep.ID))
		b.WriteByte(' ')
		b.WriteString(priorityToken(dep.Priority))
		b.WriteByte(' ')
		b.WriteString(Escape(dep.Title))
	}
	return b.String()
}

// Comments returns a ## COMMENTS section with author+timestamp headers
// and blockquoted text. Returns "" if comments is empty.
func Comments(comments []issue.Comment) string {
	if len(comments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## COMMENTS")
	for _, c := range comments {
		b.WriteString("\n\n")
		b.WriteString("**")
		b.WriteString(c.Timestamp)
		if c.Author != "" {
			b.WriteByte(' ')
			b.WriteString(c.Author)
		}
		b.WriteString("**")
		b.WriteString("\n> ")
		b.WriteString(strings.ReplaceAll(Escape(c.Text), "\n", "\n> "))
	}
	return b.String()
}

// FormatDeps returns inline dependency tokens for an issue.
// Returns "" when there are no dependencies.
func FormatDeps(iss *issue.Issue) string {
	return formatDepsInner(iss.Blocks, iss.BlockedBy, nil)
}

// FormatDepsFiltered is like FormatDeps but filters out closed blockers.
func FormatDepsFiltered(iss *issue.Issue, closedSet map[string]bool) string {
	return formatDepsInner(iss.Blocks, iss.BlockedBy, closedSet)
}

func formatDepsInner(blocks, blockedBy []string, closedSet map[string]bool) string {
	var parts []string
	if len(blocks) > 0 {
		parts = append(parts, " "+depToken("blocks", strings.Join(blocks, ",")))
	}
	var openBlockedBy []string
	for _, id := range blockedBy {
		if closedSet != nil && closedSet[id] {
			continue
		}
		openBlockedBy = append(openBlockedBy, id)
	}
	if len(openBlockedBy) > 0 {
		parts = append(parts, " "+depToken("blocked_by", strings.Join(openBlockedBy, ",")))
	}
	return strings.Join(parts, "")
}

// Token constructors

func statusToken(status string) string {
	return fmt.Sprintf("{status:%s}", status)
}

func idToken(id string) string {
	return fmt.Sprintf("{id:%s}", id)
}

func priorityToken(priority int) string {
	return fmt.Sprintf("{p:%d}", priority)
}

func typeToken(typ string) string {
	return fmt.Sprintf("{type:%s}", typ)
}

func depToken(kind, id string) string {
	return fmt.Sprintf("{dep:%s:%s}", kind, id)
}
