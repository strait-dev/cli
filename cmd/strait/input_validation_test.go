package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandEntryRejectsMalformedIDs runs every new command that takes an
// id-or-slug argument with each known malformed input and asserts:
//   - cmd.Execute returns a non-nil error.
//   - The test server is never hit (validation must happen before any HTTP call).
//
// This is defense-in-depth on top of the client-layer guard added in Fix #1.
// The command-entry guard produces friendlier user-facing errors and avoids
// wasted server roundtrips.
func TestCommandEntryRejectsMalformedIDs(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		t.Errorf("server should never be hit; got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	// Vectors that must be rejected by either ResourceID or SlugOrID.
	// (SlugOrID is stricter — it requires UUID or slug pattern — so any string
	// that doesn't match either format is rejected.)
	poisoned := []string{
		"",
		"..",
		"../etc",
		"a/b",
		"abc\x00def",
		"abc\ndef",
		"abc\rdef",
		"abc def",                // whitespace
		"UPPER",                  // not a slug, not a UUID
		"%2e%2e",                 // percent-encoded
		strings.Repeat("a", 257), // length > maxIDLen
	}

	type cmdCase struct {
		name     string
		buildCmd func(state *appState) *cobra.Command
		argFn    func(id string) []string
	}

	cases := []cmdCase{
		// Webhooks
		{name: "webhooks get", buildCmd: newWebhooksGetCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "webhooks delete", buildCmd: newWebhooksDeleteCommand, argFn: func(id string) []string { return []string{id, "--yes"} }},
		{name: "webhooks rotate-secret", buildCmd: newWebhooksRotateSecretCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "webhooks retry", buildCmd: newWebhooksRetryCommand, argFn: func(id string) []string { return []string{id} }},

		// Log drains
		{name: "log-drains get", buildCmd: newLogDrainsGetCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "log-drains update", buildCmd: newLogDrainsUpdateCommand, argFn: func(id string) []string { return []string{id, "--name", "x"} }},
		{name: "log-drains delete", buildCmd: newLogDrainsDeleteCommand, argFn: func(id string) []string { return []string{id, "--yes"} }},

		// Workflow runs
		{name: "workflow-runs pause", buildCmd: newWorkflowRunsPauseCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "workflow-runs resume", buildCmd: newWorkflowRunsResumeCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "workflow-runs retry", buildCmd: newWorkflowRunsRetryCommand, argFn: func(id string) []string { return []string{id} }},

		// Runs extras
		{name: "runs reschedule", buildCmd: newRunsRescheduleCommand, argFn: func(id string) []string { return []string{id, "--at", "2026-01-01T00:00:00Z"} }},
		{name: "runs dlq-replay", buildCmd: newRunsDLQReplayCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "runs outputs", buildCmd: newRunsOutputsCommand, argFn: func(id string) []string { return []string{id} }},
		{name: "runs checkpoints", buildCmd: newRunsCheckpointsCommand, argFn: func(id string) []string { return []string{id} }},

		// Team policies
		{name: "team policies delete", buildCmd: newTeamPoliciesDeleteCommand, argFn: func(id string) []string { return []string{id, "--yes"} }},
	}

	for _, tc := range cases {
		for _, p := range poisoned {
			t.Run(tc.name+"/"+sanitizeArgName(p), func(t *testing.T) {
				state := newTestState(t, srv)
				cmd := tc.buildCmd(state)
				cmd.SetArgs(tc.argFn(p))
				cmd.SetOut(new(strings.Builder))
				cmd.SetErr(new(strings.Builder))
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true

				err := cmd.Execute()
				if err == nil {
					t.Fatalf("%s with poisoned arg %q: expected error, got nil", tc.name, p)
				}
			})
		}
	}

	if got := hits.Load(); got != 0 {
		t.Fatalf("test server received %d unexpected hits — input validation failed", got)
	}
}

// TestStepCommandsRejectMalformedStepRef covers the two-arg workflow-runs
// step commands. Either runID or stepRef being malformed must reject the call.
func TestStepCommandsRejectMalformedStepRef(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should never be hit; got %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	state := newTestState(t, srv)

	type stepCmd struct {
		name string
		fn   func() *cobra.Command
		args func(run, step string) []string
	}

	stepCmds := []stepCmd{
		{name: "approve-step", fn: func() *cobra.Command { return newWorkflowRunsApproveStepCommand(state) }, args: func(r, s string) []string { return []string{r, s} }},
		{name: "retry-step", fn: func() *cobra.Command { return newWorkflowRunsRetryStepCommand(state) }, args: func(r, s string) []string { return []string{r, s} }},
		{name: "skip-step", fn: func() *cobra.Command { return newWorkflowRunsSkipStepCommand(state) }, args: func(r, s string) []string { return []string{r, s} }},
		{name: "force-complete-step", fn: func() *cobra.Command { return newWorkflowRunsForceCompleteStepCommand(state) }, args: func(r, s string) []string { return []string{r, s, "--yes"} }},
	}

	scenarios := []struct {
		name string
		run  string
		step string
	}{
		{name: "poisoned run id", run: "../events", step: "step-1"},
		{name: "poisoned step ref", run: "550e8400-e29b-41d4-a716-446655440000", step: "../skip"},
		{name: "control char run", run: "abc\ndef", step: "step-1"},
		{name: "control char step", run: "550e8400-e29b-41d4-a716-446655440000", step: "abc\x00"},
		{name: "empty run", run: "", step: "step-1"},
		{name: "empty step", run: "550e8400-e29b-41d4-a716-446655440000", step: ""},
	}

	for _, sc := range stepCmds {
		for _, scn := range scenarios {
			t.Run(sc.name+"/"+scn.name, func(t *testing.T) {
				cmd := sc.fn()
				cmd.SetArgs(sc.args(scn.run, scn.step))
				cmd.SetOut(new(strings.Builder))
				cmd.SetErr(new(strings.Builder))
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				if err := cmd.Execute(); err == nil {
					t.Fatalf("%s(%q,%q): expected error, got nil", sc.name, scn.run, scn.step)
				}
			})
		}
	}
}

// sanitizeArgName produces a test-name-safe label for an arbitrary string.
func sanitizeArgName(s string) string {
	if s == "" {
		return "empty"
	}
	if len(s) > 32 {
		return "long"
	}
	r := strings.NewReplacer(
		"/", "_slash_",
		`\`, "_bslash_",
		".", "_dot_",
		"\x00", "_nul_",
		"\n", "_lf_",
		"\r", "_cr_",
		"\t", "_tab_",
		" ", "_",
		"%", "_pct_",
	)
	return r.Replace(s)
}
