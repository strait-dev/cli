package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func testJobRunForExtras() types.JobRun {
	return types.JobRun{
		ID:        "run-1",
		JobID:     "job-1",
		ProjectID: "proj-test",
		Status:    "queued",
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestRunsReschedule_RequiresAt(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRunsRescheduleCommand(state)
	cmd.SetArgs([]string{"run-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --at missing")
	}
}

func TestRunsReschedule_RejectsInvalidTimestamp(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRunsRescheduleCommand(state)
	cmd.SetArgs([]string{"run-1", "--at", "tomorrow"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("expected RFC3339 error, got: %v", err)
	}
}

func TestRunsReschedule_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/reschedule": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobRunForExtras())
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsRescheduleCommand(state)
	cmd.SetArgs([]string{"run-1", "--at", "2026-12-01T00:00:00Z"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunsDLQ_Success(t *testing.T) {
	t.Parallel()

	items := []types.DLQRun{
		{ID: "dlq-1", RunID: "run-1", JobID: "job-1", Reason: "timeout", FailedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), AttemptCnt: 3},
	}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/dlq": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, items)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsDLQCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "dlq-1") {
		t.Fatalf("expected dlq id in output: %s", out)
	}
}

func TestRunsDLQReplay_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/dlq-1/dlq-replay": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobRunForExtras())
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsDLQReplayCommand(state)
	cmd.SetArgs([]string{"dlq-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunsOutputs_Success(t *testing.T) {
	t.Parallel()

	outs := []types.RunOutput{{ID: "out-1", RunID: "run-1", Key: "result"}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/outputs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, outs)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsOutputsCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "out-1") {
		t.Fatalf("expected output id in result: %s", out)
	}
}

func TestRunsToolCalls_Success(t *testing.T) {
	t.Parallel()

	calls := []types.RunToolCall{{ID: "call-1", RunID: "run-1", Tool: "search", DurationMS: 42}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/tool-calls": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, calls)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsToolCallsCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "search") {
		t.Fatalf("expected tool call in result: %s", out)
	}
}

func TestRunsCheckpoints_Success(t *testing.T) {
	t.Parallel()

	cps := []types.RunCheckpoint{{ID: "cp-1", RunID: "run-1", Name: "after-fetch"}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/checkpoints": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, cps)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsCheckpointsCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "cp-1") {
		t.Fatalf("expected checkpoint id in output: %s", out)
	}
}
