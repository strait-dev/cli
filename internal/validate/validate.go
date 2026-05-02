// Package validate provides lightweight input validation helpers used at CLI
// command entry points to catch malformed arguments before they reach the API.
package validate

import (
	"fmt"
	"regexp"
	"strings"
)

// maxIDLen is the maximum accepted length for any resource identifier.
const maxIDLen = 256

var (
	// reUUID matches the standard 8-4-4-4-12 UUID format (case-insensitive).
	reUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// reSlug matches lowercase alphanumeric strings with internal hyphens.
	reSlug = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// ResourceID validates a server-assigned resource identifier (typically a UUID).
// It rejects empty strings, strings exceeding maxIDLen characters, strings
// containing whitespace or control characters, and strings with shell-special
// characters that could indicate injection attempts.
func ResourceID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("resource ID is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("resource ID must be at most %d characters", maxIDLen)
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("resource ID contains invalid control character")
		}
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("resource ID must not contain whitespace")
	}
	return nil
}

// SlugOrID validates a CLI argument that may be either a job/workflow slug or a
// server-assigned UUID. Valid inputs are:
//   - A UUID (8-4-4-4-12 hex, case-insensitive)
//   - A slug (lowercase alphanumeric, internal hyphens only)
//
// The function does not enforce which format is used — it only rejects values
// that are neither, which are likely typos or injection attempts.
func SlugOrID(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("slug or ID is required")
	}
	if len(s) > maxIDLen {
		return fmt.Errorf("slug or ID must be at most %d characters", maxIDLen)
	}
	if reUUID.MatchString(s) {
		return nil
	}
	if reSlug.MatchString(s) {
		return nil
	}
	return fmt.Errorf("invalid slug or ID %q: must be a UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx) or a slug (lowercase letters, numbers, and hyphens only)", s)
}

// ProjectID validates a project ID, which follows the same rules as SlugOrID.
func ProjectID(id string) error {
	if err := SlugOrID(id); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	return nil
}

// IsUUID reports whether s matches the 8-4-4-4-12 hex UUID format
// (case-insensitive). Useful in resolvers to skip a speculative GET when the
// argument is clearly a slug rather than an ID.
func IsUUID(s string) bool {
	return reUUID.MatchString(strings.TrimSpace(s))
}
