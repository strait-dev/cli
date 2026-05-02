package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testRun = types.JobRun{
	ID:          "run-1",
	JobID:       "job-1",
	ProjectID:   "proj-test",
	Status:      types.StatusCompleted,
	Attempt:     1,
	Payload:     json.RawMessage(`{"key":"value"}`),
	TriggeredBy: "api",
	CreatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

var testRun2 = types.JobRun{
	ID:          "run-2",
	JobID:       "job-1",
	ProjectID:   "proj-test",
	Status:      types.StatusFailed,
	Attempt:     2,
	Payload:     json.RawMessage(`{"key":"other"}`),
	Error:       "timeout exceeded",
	TriggeredBy: "api",
	CreatedAt:   time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
}

func TestRunsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{testRun})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestRunsList_StatusFilter(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "status", "executing")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{testRun})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--status", "executing"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestRunsGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, testRun)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsGetCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestRunsCancel_SingleRun(t *testing.T) {
	t.Parallel()

	canceledRun := testRun
	canceledRun.Status = types.StatusCanceled

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/runs/run-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, canceledRun)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsCancelCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestRunsReplay_Success(t *testing.T) {
	t.Parallel()

	var replayCalls int
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/replay": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			replayCalls++
			respondJSON(t, w, http.StatusOK, types.JobRun{
				ID:          "run-replay-1",
				JobID:       "job-1",
				ParentRunID: "run-1",
				Status:      types.StatusQueued,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsReplayCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if replayCalls != 1 {
		t.Fatalf("expected exactly one replay call, got %d", replayCalls)
	}
	if !strings.Contains(out, "run-replay-1") {
		t.Fatalf("expected run-replay-1 in output, got: %s", out)
	}
	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected parent run-1 lineage in output, got: %s", out)
	}
}

func TestRunsCancel_BulkRuns(t *testing.T) {
	t.Parallel()

	var bulkBody map[string]any
	var bulkCalls int
	var deleteCalls int
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/bulk-cancel": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			bulkCalls++
			readJSONBody(t, r, &bulkBody)
			respondJSON(t, w, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"id": "run-1", "canceled": true, "status": "canceled"},
					{"id": "run-2", "canceled": true, "status": "canceled"},
				},
				"total":    2,
				"canceled": 2,
			})
		},
		"DELETE /v1/runs/run-1": func(_ http.ResponseWriter, _ *http.Request) {
			deleteCalls++
		},
		"DELETE /v1/runs/run-2": func(_ http.ResponseWriter, _ *http.Request) {
			deleteCalls++
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsCancelCommand(state)
	cmd.SetArgs([]string{"run-1", "run-2", "--yes"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if bulkCalls != 1 {
		t.Fatalf("expected exactly one bulk-cancel call, got %d", bulkCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("expected no per-run DELETE calls, got %d", deleteCalls)
	}
	ids, ok := bulkBody["ids"].([]any)
	if !ok || len(ids) != 2 {
		t.Fatalf("expected 2 ids in bulk body, got: %v", bulkBody)
	}
	if !strings.Contains(out, "run-1") || !strings.Contains(out, "run-2") {
		t.Fatalf("expected run-1 and run-2 in output, got: %s", out)
	}
}

func TestRunsLast_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "limit", "1")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{testRun})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsLastCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestRunsLast_NoRuns(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.JobRun{})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsLastCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no runs found") {
		t.Fatalf("expected 'no runs found' error, got: %v", err)
	}
}

func TestRunsDiff_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, testRun)
		},
		"GET /v1/runs/run-2": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, testRun2)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsDiffCommand(state)
	cmd.SetArgs([]string{"run-1", "run-2"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
	if !strings.Contains(out, "run-2") {
		t.Fatalf("expected run-2 in output, got: %s", out)
	}
}
