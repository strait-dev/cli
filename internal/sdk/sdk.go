// Package sdk is a thin shim that the CLI uses to construct the canonical
// strait Go SDK client (github.com/strait-dev/strait-go) once it is published.
//
// Until strait-go v0.2.0 lands the shim returns the existing
// internal/client.Client as the underlying transport. New CLI commands should
// depend on this package rather than internal/client directly so the eventual
// swap is a single edit.
//
// See internal/AGENTS.md for the migration policy (when to call sdk.New vs.
// internal/client.New).
package sdk

import (
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/config"
)

// Client is the orchestration client used by new CLI commands. While the Go
// SDK is unpublished it is a thin alias around *internal/client.Client. When
// strait-go v0.2.0 ships, the alias will be replaced with *strait.Client and
// the constructor will route to strait.NewClient(...).
type Client = client.Client

// New constructs an orchestration client from the resolved CLI configuration.
//
// New commands (worker, endpoint, deploy push) should call this. Existing
// commands continue to call internal/client.New until they are individually
// migrated; do not re-implement an API that already lives on the internal
// client.
func New(resolved config.Resolved) (*Client, error) {
	timeout := 30 * time.Second
	if resolved.Timeout != "" {
		if d, err := time.ParseDuration(resolved.Timeout); err == nil && d > 0 {
			timeout = d
		}
	}
	return client.New(resolved.ServerURL, resolved.Credential, timeout)
}
