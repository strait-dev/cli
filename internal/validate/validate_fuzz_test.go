package validate_test

import (
	"testing"

	"github.com/strait-dev/cli/internal/validate"
)

// FuzzResourceID ensures ResourceID never panics on arbitrary input.
func FuzzResourceID(f *testing.F) {
	f.Add("")
	f.Add("abc")
	f.Add("123e4567-e89b-12d3-a456-426614174000")
	f.Add("my-job-slug")
	f.Add(string([]byte{0x00, 0x01, 0x1f}))
	f.Add("$(echo injection)")
	f.Add("../../../../etc/passwd")

	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic.
		_ = validate.ResourceID(s)
	})
}

// FuzzSlugOrID ensures SlugOrID never panics on arbitrary input.
func FuzzSlugOrID(f *testing.F) {
	f.Add("")
	f.Add("my-job")
	f.Add("123e4567-e89b-12d3-a456-426614174000")
	f.Add("HasUpperCase")
	f.Add("has space")
	f.Add("$(rm -rf /)")
	f.Add("\x00\x01\x02")
	f.Add("a\tb")

	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic.
		_ = validate.SlugOrID(s)
	})
}
