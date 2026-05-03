package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func testWorkflowRunForExtras() types.WorkflowRun {
	return types.WorkflowRun{
		ID:         "wfrun-1",
		WorkflowID: "wf-1",
		ProjectID:  "proj-test",
		Status:     "running",
		CreatedAt:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestWorkflowRunsPause_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/pause": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWorkflowRunForExtras())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsPauseCommand(state)
	cmd.SetArgs([]string{"wfrun-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsResume_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/resume": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWorkflowRunForExtras())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsResumeCommand(state)
	cmd.SetArgs([]string{"wfrun-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsRetry_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/retry": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWorkflowRunForExtras())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsRetryCommand(state)
	cmd.SetArgs([]string{"wfrun-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsApproveStep_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/steps/step-a/approve": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsApproveStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "step-a"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsApproveStep_RejectsInvalidStepRef(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsApproveStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "BAD STEP"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid step ref") {
		t.Fatalf("expected step ref error, got: %v", err)
	}
}

func TestWorkflowRunsRetryStep_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/steps/step-a/retry": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsRetryStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "step-a"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsSkipStep_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/steps/step-a/skip": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsSkipStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "step-a"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsForceCompleteStep_RequiresYes(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsForceCompleteStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "step-a"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestWorkflowRunsForceCompleteStep_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-1/steps/step-a/force-complete": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowRunsForceCompleteStepCommand(state)
	cmd.SetArgs([]string{"wfrun-1", "step-a", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
