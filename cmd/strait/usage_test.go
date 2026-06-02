package main

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// TestUsageCommand_BareInvocationCallsCurrent asserts that `strait usage`
// (no sub-command) routes to the same handler as `strait usage current`.
// Previously this was wired via a shim that constructed a fresh cobra.Command
// inside RunE just to reach its function pointer — wasteful and obscure.
// The free-function refactor must preserve the routing behavior.
func TestUsageCommand_BareInvocationCallsCurrent(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC)
	period := types.UsagePeriod{
		PeriodStart:    start,
		PeriodEnd:      end,
		Runs:           42,
		WorkflowRuns:   7,
		ComputeMinutes: 13.5,
		CostUSD:        9.99,
	}

	var hits atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/usage/current": func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			respondJSON(t, w, http.StatusOK, period)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"
	cmd := newUsageCommand(state)
	cmd.SetArgs([]string{}) // bare invocation

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 hit on /v1/usage/current, got %d", hits.Load())
	}

	var got types.UsagePeriod
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\noutput: %s", err, out)
	}
	if got.Runs != period.Runs || got.CostUSD != period.CostUSD {
		t.Fatalf("payload mismatch: got %+v, want %+v", got, period)
	}
}

// TestUsageCommand_CurrentSubcommandReachesSameEndpoint pairs with the test
// above to confirm the explicit sub-command form behaves identically.
func TestUsageCommand_CurrentSubcommandReachesSameEndpoint(t *testing.T) {
	t.Parallel()

	period := types.UsagePeriod{Runs: 1, CostUSD: 0.01}

	var hits atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/usage/current": func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			respondJSON(t, w, http.StatusOK, period)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"
	cmd := newUsageCommand(state)
	cmd.SetArgs([]string{"current"})

	if _, err := captureCommandOutputErr(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 hit, got %d", hits.Load())
	}
}

// captureCommandOutputErr is a tiny shim around cmd.Execute that returns the
// error so subtests can assert it directly. Output is intentionally discarded.
func captureCommandOutputErr(cmd interface{ Execute() error }) (string, error) {
	return "", cmd.Execute()
}
