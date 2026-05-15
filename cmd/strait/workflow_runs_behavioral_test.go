package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testWorkflowRun = types.WorkflowRun{
	ID:              "wfr-1",
	WorkflowID:      "wf-1",
	ProjectID:       "proj-test",
	Status:          types.WfStatusRunning,
	TriggeredBy:     "manual",
	WorkflowVersion: 1,
	CreatedAt:       time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

var testStepRun = types.WorkflowStepRun{
	ID:             "sr-1",
	WorkflowRunID:  "wfr-1",
	WorkflowStepID: "step-1",
	StepRef:        "process",
	Attempt:        1,
	Status:         types.StepCompleted,
	CreatedAt:      time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

func TestWorkflowRunsList_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.WorkflowRun{testWorkflowRun})
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "wfr-1") {
		t.Fatalf("expected wfr-1 in output, got: %s", out)
	}
}

func TestWorkflowRunsList_StatusFilter(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "status", "completed")
			respondPaginated(t, w, http.StatusOK, []types.WorkflowRun{})
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--status", "completed"})
	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWorkflowRunsGet_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfr-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWorkflowRun)
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsGetCommand(state)
	cmd.SetArgs([]string{"wfr-1"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "wfr-1") {
		t.Fatalf("expected wfr-1 in output, got: %s", out)
	}
}

func TestWorkflowRunsCancel_Success(t *testing.T) {
	t.Parallel()
	cancelCalled := false
	canceled := testWorkflowRun
	canceled.Status = types.WfStatusCanceled
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/workflow-runs/wfr-1": func(w http.ResponseWriter, _ *http.Request) {
			cancelCalled = true
			respondJSON(t, w, http.StatusOK, canceled)
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsCancelCommand(state)
	cmd.SetArgs([]string{"wfr-1"})
	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !cancelCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestWorkflowRunsWatch_Timeout(t *testing.T) {
	// Not parallel — mutates package-level workflowRunsTimeNow/After.
	originalNow := workflowRunsTimeNow
	originalAfter := workflowRunsAfter
	t.Cleanup(func() {
		workflowRunsTimeNow = originalNow
		workflowRunsAfter = originalAfter
	})

	// Step time forward on each call so the deadline is reached after one poll.
	calls := 0
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	workflowRunsTimeNow = func() time.Time {
		calls++
		return base.Add(time.Duration(calls) * time.Minute)
	}
	workflowRunsAfter = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- base
		return ch
	}

	running := testWorkflowRun
	running.Status = types.WfStatusRunning
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfr-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, running)
		},
		"GET /v1/workflow-runs/wfr-1/steps": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.WorkflowStepRun{testStepRun})
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsWatchCommand(state)
	cmd.SetArgs([]string{"wfr-1", "--timeout", "10s", "--interval", "1ms"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout reached") {
		t.Fatalf("expected timeout reached, got: %v", err)
	}
}

func TestWorkflowRunsWatch_TerminalCompleted(t *testing.T) {
	// Not parallel — mutates package-level workflowRunsAfter.
	originalAfter := workflowRunsAfter
	t.Cleanup(func() { workflowRunsAfter = originalAfter })
	workflowRunsAfter = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	done := testWorkflowRun
	done.Status = types.WfStatusCompleted
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfr-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, done)
		},
		"GET /v1/workflow-runs/wfr-1/steps": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.WorkflowStepRun{testStepRun})
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsWatchCommand(state)
	cmd.SetArgs([]string{"wfr-1", "--timeout", "5s", "--interval", "1ms"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success on terminal completed, got: %v", err)
	}
}

func TestWorkflowRunsSteps_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfr-1/steps": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.WorkflowStepRun{testStepRun})
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsStepsListCommand(state)
	cmd.SetArgs([]string{"wfr-1"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "sr-1") {
		t.Fatalf("expected sr-1 in output, got: %s", out)
	}
}
