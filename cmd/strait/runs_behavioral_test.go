package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
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

func TestRunsGet_TTYIncludesOptionalFields(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC)
	finishedAt := startedAt.Add(3 * time.Minute)
	run := testRun
	run.StartedAt = &startedAt
	run.FinishedAt = &finishedAt
	run.Error = "worker exited with status 1"

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, run)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newRunsGetCommand(state)
		cmd.SetArgs([]string{"run-1"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Started", "Finished", "Error", "worker exited with status 1"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}

func TestRunsCancel_AllRequiresProject(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{
		serverURL:    "http://127.0.0.1",
		apiKey:       "test-key",
		outputFormat: "json",
		ciMode:       true,
	}}
	cmd := newRunsCancelCommand(state)
	cmd.SetArgs([]string{"--all"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected missing project error, got: %v", err)
	}
}

func TestRunsCancel_AllReturnsListError(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusBadGateway, "runs backend unavailable")
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsCancelCommand(state)
	cmd.SetArgs([]string{"--all", "--project", "proj-test"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "request failed (502)") {
		t.Fatalf("expected list error, got: %v", err)
	}
}

func TestRunsCancel_AllTTYReportsMixedResults(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "limit", "100")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", JobID: "job-1", ProjectID: "proj-test", Status: types.StatusExecuting},
				{ID: "run-2", JobID: "job-1", ProjectID: "proj-test", Status: types.StatusExecuting},
			})
		},
		"POST /v1/runs/bulk-cancel": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, client.BulkCancelRunsResponse{
				Results: []client.BulkCancelResult{
					{ID: "run-1", Canceled: true, Status: "canceled"},
					{ID: "run-2", Canceled: false, Error: "already terminal"},
				},
				Total:    2,
				Canceled: 1,
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newRunsCancelCommand(state)
		cmd.SetArgs([]string{"--all", "--project", "proj-test", "--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Canceled run", "run-1", "Failed to cancel run-2", "already terminal"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}

func TestRunsLogs_NonFollowPrintsFilteredRows(t *testing.T) {
	t.Parallel()

	eventTime := time.Date(2026, 3, 20, 10, 1, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "level", "error")
			assertQuery(t, r, "type", "error")
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{{
				ID:        "evt-1",
				RunID:     "run-1",
				Type:      types.EventType("error"),
				Level:     "error",
				Message:   "job failed",
				CreatedAt: eventTime,
			}})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsLogsCommand(state)
	cmd.SetArgs([]string{"run-1", "--level", "error", "--type", "error"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job failed") || !strings.Contains(out, "run-1") {
		t.Fatalf("expected filtered log output, got: %s", out)
	}
}

func TestRunsLogs_FollowPrintsExistingAndStreamedRows(t *testing.T) {
	t.Parallel()

	eventTime := time.Date(2026, 3, 20, 10, 1, 0, 0, time.UTC)
	streamTime := eventTime.Add(time.Minute)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusExecuting})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{{
				ID:        "evt-1",
				RunID:     "run-1",
				Type:      types.EventType("log"),
				Level:     "info",
				Message:   "existing log line",
				CreatedAt: eventTime,
			}})
		},
		"GET /v1/runs/run-1/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: event\n"))
			_, _ = w.Write([]byte(`data: {"timestamp":"` + streamTime.Format(time.RFC3339Nano) + `","level":"warn","event_type":"log","message":"streamed log line"}` + "\n\n"))
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsLogsCommand(state)
	cmd.SetArgs([]string{"run-1", "--follow"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"existing log line", "streamed log line", "\"level\":\"warn\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestRunsReplay_TTYWaitsForCompletion(t *testing.T) {
	t.Parallel()

	var replayPolls int
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/replay": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id":     "run-replay-1",
				"status": "queued",
			})
		},
		"GET /v1/runs/run-replay-1": func(w http.ResponseWriter, _ *http.Request) {
			replayPolls++
			respondJSON(t, w, http.StatusOK, types.JobRun{
				ID:      "run-replay-1",
				Status:  types.StatusCompleted,
				Attempt: 1,
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newRunsReplayCommand(state)
		cmd.SetArgs([]string{"run-1", "--wait"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if replayPolls == 0 {
		t.Fatal("expected replay wait to poll the replayed run")
	}
	for _, want := range []string{"Replayed as run", "run-replay-1", "Run completed"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}
