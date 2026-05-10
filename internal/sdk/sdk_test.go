package sdk

import (
	"testing"

	"github.com/strait-dev/cli/internal/config"
)

func TestNew_ReturnsClient(t *testing.T) {
	t.Parallel()

	cli, err := New(config.Resolved{
		ServerURL:  "https://api.example.com",
		Credential: "test-key",
		Timeout:    "5s",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNew_DefaultsTimeoutOnInvalid(t *testing.T) {
	t.Parallel()

	cli, err := New(config.Resolved{
		ServerURL:  "https://api.example.com",
		Credential: "test-key",
		Timeout:    "not-a-duration",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil client")
	}
}
