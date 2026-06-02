package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// buildAnalyticsCoverageParent constructs a bare analytics parent command and
// registers all coverage sub-commands on it, mirroring how a real command tree
// would be assembled before registerAnalyticsCoverageCommands is wired into
// newAnalyticsCommand.
func buildAnalyticsCoverageParent(state *appState) *cobra.Command {
	parent := &cobra.Command{
		Use:   "analytics",
		Short: "analytics (test parent)",
	}
	registerAnalyticsCoverageCommands(parent, state)
	return parent
}

// execSub sets args on parent and executes it, returning any error.
func execSub(parent *cobra.Command, args ...string) error {
	parent.SetArgs(args)
	return parent.Execute()
}

// TestAnalyticsCoverage_Approvals verifies that the approvals sub-command sends
// GET /v1/analytics/approvals with the expected project_id query parameter.
func TestAnalyticsCoverage_Approvals(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/approvals": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodGet)
			assertQuery(t, r, "project_id", "proj-test")
			respondJSON(t, w, http.StatusOK, map[string]any{"total": 5})
		},
	})

	state := newTestState(t, srv)
	parent := buildAnalyticsCoverageParent(state)

	out := captureStateOutput(t, state, func() {
		if err := execSub(parent, "approvals", "--project", "proj-test"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "total") {
		t.Fatalf("expected response body in output, got: %s", out)
	}
}

// TestAnalyticsCoverage_RunsSummary verifies that the runs summary sub-command
// sends GET /v1/analytics/runs/summary with project_id and optional window
// params.
func TestAnalyticsCoverage_RunsSummary(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/runs/summary": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodGet)
			assertQuery(t, r, "project_id", "proj-test")
			if got := r.URL.Query().Get("from"); got != "2026-01-01T00:00:00Z" {
				t.Errorf("from: got %q, want %q", got, "2026-01-01T00:00:00Z")
			}
			if got := r.URL.Query().Get("to"); got != "2026-02-01T00:00:00Z" {
				t.Errorf("to: got %q, want %q", got, "2026-02-01T00:00:00Z")
			}
			respondJSON(t, w, http.StatusOK, map[string]any{"total_runs": 42})
		},
	})

	state := newTestState(t, srv)
	parent := buildAnalyticsCoverageParent(state)

	out := captureStateOutput(t, state, func() {
		if err := execSub(parent, "runs", "summary",
			"--project", "proj-test",
			"--from", "2026-01-01T00:00:00Z",
			"--to", "2026-02-01T00:00:00Z",
		); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "total_runs") {
		t.Fatalf("expected total_runs in output, got: %s", out)
	}
}

// TestAnalyticsCoverage_JobsHistory verifies that the jobs history sub-command
// sends GET /v1/analytics/jobs/{jobID}/history with the correct path and
// project_id query parameter.
func TestAnalyticsCoverage_JobsHistory(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/jobs/job-abc/history": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodGet)
			assertQuery(t, r, "project_id", "proj-test")
			respondJSON(t, w, http.StatusOK, map[string]any{"job_id": "job-abc", "runs": 10})
		},
	})

	state := newTestState(t, srv)
	parent := buildAnalyticsCoverageParent(state)

	out := captureStateOutput(t, state, func() {
		if err := execSub(parent, "jobs", "history", "job-abc", "--project", "proj-test"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-abc") {
		t.Fatalf("expected job-abc in output, got: %s", out)
	}
}

// TestAnalyticsCoverage_TagsCost verifies that the tags cost sub-command sends
// GET /v1/analytics/tags/cost with the expected project_id query parameter.
func TestAnalyticsCoverage_TagsCost(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/tags/cost": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodGet)
			assertQuery(t, r, "project_id", "proj-test")
			respondJSON(t, w, http.StatusOK, map[string]any{"tags": []string{"production", "staging"}})
		},
	})

	state := newTestState(t, srv)
	parent := buildAnalyticsCoverageParent(state)

	out := captureStateOutput(t, state, func() {
		if err := execSub(parent, "tags", "cost", "--project", "proj-test"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "tags") {
		t.Fatalf("expected tags in output, got: %s", out)
	}
}
