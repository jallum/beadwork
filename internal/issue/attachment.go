package issue

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// GetAttachment returns the bytes stored at
// attachments/<ticketID>/<path> in the current Beadwork tree. Paths
// are looked up verbatim, matching the storage layout described in
// docs/design.md. Returns an error wrapping ErrAttachmentNotFound when
// the path is absent.
func (s *Store) GetAttachment(ticketID string, path string) ([]byte, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket id is empty")
	}
	if err := validateAttachmentPath(path); err != nil {
		return nil, err
	}
	p := attachmentPath(ticketID, path)
	data, err := s.FS.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAttachmentNotFound, p)
	}
	return data, nil
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

// ReadAttachmentSource returns the bytes of an attachment blob, looking
// first in the current TreeFS overlay/base and then falling back to
// SourceHash when set. Returns an error wrapping fs.ErrNotExist when
// the attachment is unreachable from either source. Used by the intent
// replay handler for the `attach` verb.
func (s *Store) ReadAttachmentSource(ticketID, storedPath string) ([]byte, error) {
	p := attachmentPath(ticketID, storedPath)
	if data, err := s.FS.ReadFile(p); err == nil {
		return data, nil
	}
	if !s.SourceHash.IsZero() {
		data, err := s.FS.ReadFileAt(s.SourceHash, p)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read attachment from source tree: %w", err)
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrAttachmentNotFound, p)
}
