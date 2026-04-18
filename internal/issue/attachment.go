package issue

import (
	"fmt"
	"strings"
)

// ErrAttachmentNotFound is returned by GetAttachment when the requested
// attachment path is absent from the Beadwork tree.
var ErrAttachmentNotFound = fmt.Errorf("attachment not found")

// attachmentsRoot is the top-level directory under which attachment blobs
// are stored. Layout is attachments/<ticket-id>/<path-verbatim>. See
// docs/design.md for the full format.
const attachmentsRoot = "attachments"

// validateAttachmentPath rejects paths that would embed a newline or
// carry trailing whitespace. The stored path is otherwise passed through
// verbatim so the on-disk layout matches the Elixir port bit-for-bit.
func validateAttachmentPath(p string) error {
	if p == "" {
		return fmt.Errorf("attachment path is empty")
	}
	if strings.ContainsAny(p, "\n\r") {
		return fmt.Errorf("attachment path contains newline: %q", p)
	}
	if strings.TrimRight(p, " \t") != p {
		return fmt.Errorf("attachment path has trailing whitespace: %q", p)
	}
	return nil
}

// Attach writes content as a blob under attachments/<ticketID>/<storedPath>.
// It stages the tree entry via the existing TreeFS helpers; the caller is
// responsible for forming the matching `attach <ticketID> <storedPath>`
// intent line and calling Commit (see docs/design.md for the grammar).
//
// Paths are stored verbatim: no normalization, no basename flattening.
// Nested paths (containing "/") are allowed.
func (s *Store) Attach(ticketID string, storedPath string, content []byte) error {
	if ticketID == "" {
		return fmt.Errorf("ticket id is empty")
	}
	if strings.ContainsAny(ticketID, " \t\n\r/") {
		return fmt.Errorf("invalid ticket id %q", ticketID)
	}
	if err := validateAttachmentPath(storedPath); err != nil {
		return err
	}
	return s.FS.WriteFile(attachmentPath(ticketID, storedPath), content)
}

// attachmentPath returns the tree path for an attachment.
func attachmentPath(ticketID, storedPath string) string {
	return attachmentsRoot + "/" + ticketID + "/" + storedPath
}
