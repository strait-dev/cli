package client

import (
	"fmt"
	"path"
	"strings"
)

// joinPath safely joins URL path segments after validating each one against
// path-traversal and smuggling vectors. The first argument is a static prefix
// that the caller controls (e.g. "/v1/jobs"); subsequent segments are treated
// as untrusted and validated.
func joinPath(prefix string, segments ...string) (string, error) {
	for _, s := range segments {
		if err := validatePathSegment(s); err != nil {
			return "", err
		}
	}
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, prefix)
	parts = append(parts, segments...)
	return path.Join(parts...), nil
}

// validatePathSegment rejects any string that would change the resolved URL
// path beyond the segment it occupies. It catches:
//   - empty / "." / ".." segments
//   - segments containing forward or back slashes
//   - segments containing control characters (0x00-0x1F, 0x7F)
//   - segments starting with "%" (already-encoded sequences are ambiguous and
//     would be re-encoded by the URL builder)
func validatePathSegment(s string) error {
	if s == "" {
		return fmt.Errorf("path segment must not be empty")
	}
	if s == "." || s == ".." {
		return fmt.Errorf("path segment %q is not allowed", s)
	}
	if strings.ContainsAny(s, "/\\") {
		return fmt.Errorf("path segment must not contain a slash: %q", s)
	}
	if strings.HasPrefix(s, "%") {
		return fmt.Errorf("path segment must not start with %% (pre-encoded sequences are not allowed)")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("path segment must not contain control characters")
		}
	}
	return nil
}
