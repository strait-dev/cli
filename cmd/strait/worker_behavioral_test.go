package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func testWorkers() []types.WorkerInfo {
	return []types.WorkerInfo{
		{
			ID:          "wkr-1",
			Name:        "worker-1",
			ProjectID:   "proj-test",
			Queues:      []string{"default"},
			Status:      "online",
			ActiveTasks: 2,
			ConnectedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "wkr-2",
			Name:        "worker-2",
			ProjectID:   "proj-test",
			Queues:      []string{"video", "audio"},
			Status:      "online",
			ActiveTasks: 0,
			ConnectedAt: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC),
		},
	}
}

func TestWorkerStatus_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workers": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, testWorkers())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkerStatusCommand(state)
	cmd.SetArgs([]string{})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wkr-1") || !strings.Contains(out, "wkr-2") {
		t.Fatalf("expected worker ids in output: %s", out)
	}
}

func TestWorkerStatus_Empty(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workers": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.WorkerInfo{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkerStatusCommand(state)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkerDrain_Success(t *testing.T) {
	t.Parallel()

	var hit bool
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workers/wkr-1/disconnect": func(w http.ResponseWriter, _ *http.Request) {
			hit = true
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "draining"})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkerDrainCommand(state)
	cmd.SetArgs([]string{"wkr-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hit {
		t.Fatal("expected disconnect endpoint to be called")
	}
	if !strings.Contains(out, "wkr-1") {
		t.Fatalf("expected worker id in output: %s", out)
	}
}

func TestWorkerCommand_Wiring(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	worker := findSubcommand(t, cmd, "worker")
	for _, sub := range []string{"status", "drain"} {
		findSubcommand(t, worker, sub)
	}
}
