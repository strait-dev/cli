package sdk

import (
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/config"
)

func TestNew_WiresBaseURLAndServices(t *testing.T) {
	t.Parallel()

	c, err := New(config.Resolved{
		ServerURL:  "https://api.example.com",
		Credential: "sk_test_123",
		Timeout:    "5s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.Client == nil {
		t.Fatal("expected wrapped strait.Client")
	}
	if got := c.BaseURL(); !strings.HasPrefix(got, "https://api.example.com") {
		t.Fatalf("base URL not wired: got %q", got)
	}
	if c.Jobs == nil || c.Workflows == nil || c.Runs == nil {
		t.Fatal("expected jobs/workflows/runs services on the wrapped client")
	}
}

func TestNew_DefaultsWhenEmpty(t *testing.T) {
	t.Parallel()

	c, err := New(config.Resolved{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.Client == nil {
		t.Fatal("expected wrapped strait.Client even with empty resolved config")
	}
}

func TestNew_IgnoresBadTimeout(t *testing.T) {
	t.Parallel()

	c, err := New(config.Resolved{
		ServerURL: "https://api.example.com",
		Timeout:   "not-a-duration",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected client even with malformed timeout")
	}
}
