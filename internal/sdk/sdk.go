// Package sdk is the canonical Go-SDK client for new CLI commands.
//
// It wraps *github.com/strait-dev/strait-go.Client so new commands target the
// published SDK surface (Jobs, Workflows, Runs, etc.) instead of internal/client.
// Existing commands continue to use internal/client until they are individually
// migrated; do not re-implement an API that already lives there.
//
// See internal/AGENTS.md for the migration policy.
package sdk

import (
	"time"

	"github.com/strait-dev/cli/internal/config"
	strait "github.com/strait-dev/strait-go"
)

// Client is the orchestration client used by new CLI commands. It embeds the
// strait-go *strait.Client so all 23 services (Jobs, Workflows, Runs, ...) are
// reachable directly: `c.Jobs.Get(ctx, slug)`.
type Client struct {
	*strait.Client
}

// New constructs an orchestration client from the resolved CLI configuration.
// resolved.Timeout, when non-empty, is parsed as a time.Duration and forwarded
// as milliseconds to strait.WithTimeout.
func New(resolved config.Resolved) (*Client, error) {
	opts := []strait.Option{}
	if resolved.ServerURL != "" {
		opts = append(opts, strait.WithBaseURL(resolved.ServerURL))
	}
	if resolved.Credential != "" {
		opts = append(opts, strait.WithAPIKey(resolved.Credential))
	}
	if resolved.Timeout != "" {
		if d, err := time.ParseDuration(resolved.Timeout); err == nil && d > 0 {
			opts = append(opts, strait.WithTimeout(int(d/time.Millisecond)))
		}
	}
	return &Client{Client: strait.NewClient(opts...)}, nil
}
