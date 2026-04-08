package validate_test

import (
	"testing"

	"github.com/strait-dev/cli/internal/validate"
)

func TestResourceID_Valid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"abc123",
		"123e4567-e89b-12d3-a456-426614174000",
		"job-abc",
		"a",
		"a1b2c3d4e5f6",
	}
	for _, id := range cases {
		if err := validate.ResourceID(id); err != nil {
			t.Errorf("ResourceID(%q) unexpected error: %v", id, err)
		}
	}
}

func TestResourceID_Invalid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		wantErr string
	}{
		{"", "required"},
		{" ", "required"},
		{string([]byte{0x01, 0x02}), "control character"},
		{"has space in it", "whitespace"},
	}
	for _, tc := range cases {
		err := validate.ResourceID(tc.input)
		if err == nil {
			t.Errorf("ResourceID(%q) expected error containing %q, got nil", tc.input, tc.wantErr)
			continue
		}
		if !contains(err.Error(), tc.wantErr) {
			t.Errorf("ResourceID(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
		}
	}
}

func TestResourceID_TooLong(t *testing.T) {
	t.Parallel()

	long := make([]byte, 257)
	for i := range long {
		long[i] = 'a'
	}
	err := validate.ResourceID(string(long))
	if err == nil {
		t.Fatal("expected error for too-long ID")
	}
}

func TestSlugOrID_ValidSlug(t *testing.T) {
	t.Parallel()

	cases := []string{
		"my-job",
		"job123",
		"a",
		"process-payments-v2",
	}
	for _, s := range cases {
		if err := validate.SlugOrID(s); err != nil {
			t.Errorf("SlugOrID(%q) unexpected error: %v", s, err)
		}
	}
}

func TestSlugOrID_ValidUUID(t *testing.T) {
	t.Parallel()

	cases := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"00000000-0000-0000-0000-000000000000",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", // uppercase accepted
	}
	for _, s := range cases {
		if err := validate.SlugOrID(s); err != nil {
			t.Errorf("SlugOrID(%q) unexpected error: %v", s, err)
		}
	}
}

func TestSlugOrID_Invalid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		" ",
		"Has-Upper",
		"has space",
		"has/slash",
		"has;semicolon",
		"$(injection)",
		"../../traversal",
	}
	for _, s := range cases {
		if err := validate.SlugOrID(s); err == nil {
			t.Errorf("SlugOrID(%q) expected error, got nil", s)
		}
	}
}

func TestSlugOrID_StartsOrEndsWithHyphen(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"-starts-with-hyphen", "ends-with-hyphen-"} {
		if err := validate.SlugOrID(s); err == nil {
			t.Errorf("SlugOrID(%q) expected error for leading/trailing hyphen", s)
		}
	}
}

func TestProjectID_Valid(t *testing.T) {
	t.Parallel()

	if err := validate.ProjectID("my-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectID_Invalid(t *testing.T) {
	t.Parallel()

	if err := validate.ProjectID(""); err == nil {
		t.Fatal("expected error for empty project ID")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
