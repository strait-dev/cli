package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/types"
)

func TestWaitRun_CompletesImmediately(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "run-1", "status": "completed", "attempt": 1,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newWaitRunCommand(state)
	cmd.SetArgs([]string{"run-1", "--for", "status=completed", "--timeout", "5s"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWaitRun_Timeout(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "run-1", "status": "executing", "attempt": 1,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newWaitRunCommand(state)
	cmd.SetArgs([]string{"run-1", "--for", "status=completed", "--timeout", "100ms", "--interval", "20ms"})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "timeout") {
			t.Fatalf("expected timeout error, got: %v", err)
		}
	})
}

func TestWaitRun_InvalidCondition(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newWaitRunCommand(state)
	cmd.SetArgs([]string{"run-1", "--for", "badcondition"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported condition") {
		t.Fatalf("expected unsupported condition error, got: %v", err)
	}
}

func TestWaitQueue_EmptyImmediately(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{
				"queued": 0, "executing": 0, "delayed": 0,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newWaitQueueCommand(state)
	cmd.SetArgs([]string{"--empty", "--timeout", "5s"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWaitQueue_RequiresEmpty(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newWaitQueueCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "only --empty") {
		t.Fatalf("expected --empty error, got: %v", err)
	}
}

func TestParseWaitCondition_AllStatuses(t *testing.T) {
	t.Parallel()

	valid := []types.RunStatus{
		types.StatusDelayed, types.StatusQueued, types.StatusDequeued,
		types.StatusExecuting, types.StatusWaiting, types.StatusCompleted,
		types.StatusFailed, types.StatusTimedOut, types.StatusCrashed,
		types.StatusSystemFailed, types.StatusCanceled, types.StatusExpired,
	}
	for _, s := range valid {
		got, err := parseWaitCondition("status=" + string(s))
		if err != nil {
			t.Errorf("parseWaitCondition(status=%s): %v", s, err)
		}
		if got != s {
			t.Errorf("expected %s, got %s", s, got)
		}
	}
}

func TestParseWaitCondition_InvalidStatus(t *testing.T) {
	t.Parallel()

	_, err := parseWaitCondition("status=bogus")
	if err == nil || !strings.Contains(err.Error(), "unsupported run status") {
		t.Fatalf("expected unsupported status error, got: %v", err)
	}
}

func TestParseWaitCondition_BadFormat(t *testing.T) {
	t.Parallel()

	_, err := parseWaitCondition("not-a-condition")
	if err == nil || !strings.Contains(err.Error(), "unsupported condition") {
		t.Fatalf("expected unsupported condition error, got: %v", err)
	}
}
