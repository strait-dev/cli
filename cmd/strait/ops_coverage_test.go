package main

import (
	"net/http"
	"strings"
	"testing"
)

// Export jobs.

func TestExportJobs_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/export/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "format", "json")
			respondJSON(t, w, http.StatusOK, map[string]any{"jobs": []string{"job-1"}})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportJobsCommand(state)
	cmd.SetArgs([]string{"--format", "json"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job id in output: %s", out)
	}
}

func TestExportJobs_DefaultFormat(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/export/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "format", "json")
			respondJSON(t, w, http.StatusOK, map[string]any{})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportJobsCommand(state)

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Export runs.

func TestExportRuns_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/export/runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "format", "json")
			assertQuery(t, r, "from", "2026-01-01T00:00:00Z")
			assertQuery(t, r, "to", "2026-06-01T00:00:00Z")
			respondJSON(t, w, http.StatusOK, map[string]any{"runs": []string{"run-1"}})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportRunsCommand(state)
	cmd.SetArgs([]string{"--format", "json", "--from", "2026-01-01T00:00:00Z", "--to", "2026-06-01T00:00:00Z"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run id in output: %s", out)
	}
}

// Export workflows.

func TestExportWorkflows_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/export/workflows": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "format", "json")
			respondJSON(t, w, http.StatusOK, map[string]any{"workflows": []string{"wf-1"}})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportWorkflowsCommand(state)
	cmd.SetArgs([]string{"--format", "json"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected workflow id in output: %s", out)
	}
}

// Stats.

func TestStats_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{"total_jobs": 42, "total_runs": 1000})
		},
	})

	state := newTestState(t, srv)
	cmd := newStatsCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "total_jobs") {
		t.Fatalf("expected stats fields in output: %s", out)
	}
}

// Batch operations list.

func TestBatchOperationsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/batch-operations": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "batch-1", "status": "completed"},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newBatchOperationsListCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "batch-1") {
		t.Fatalf("expected batch operation id in output: %s", out)
	}
}

func TestBatchOperationsList_WithLimit(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/batch-operations": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "limit", "10")
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	cmd := newBatchOperationsListCommand(state)
	cmd.SetArgs([]string{"--limit", "10"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Batch operations get.

func TestBatchOperationsGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/batch-operations/batch-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "batch-1", "status": "completed"})
		},
	})

	state := newTestState(t, srv)
	cmd := newBatchOperationsGetCommand(state)
	cmd.SetArgs([]string{"batch-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "batch-1") {
		t.Fatalf("expected batch operation id in output: %s", out)
	}
}

// Organizations jobs.

func TestOrganizationsJobs_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/organizations/org-1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "job-1", "slug": "my-job"},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newOrganizationsJobsCommand(state)
	cmd.SetArgs([]string{"org-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job id in output: %s", out)
	}
}

func TestOrganizationsJobs_WithLimit(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/organizations/org-1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "limit", "5")
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	cmd := newOrganizationsJobsCommand(state)
	cmd.SetArgs([]string{"org-1", "--limit", "5"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Organizations runs.

func TestOrganizationsRuns_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/organizations/org-1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "run-1", "status": "completed"},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newOrganizationsRunsCommand(state)
	cmd.SetArgs([]string{"org-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run id in output: %s", out)
	}
}
