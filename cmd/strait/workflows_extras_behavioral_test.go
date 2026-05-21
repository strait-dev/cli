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

func testWorkflowForExtras() client.WorkflowResponse {
	return client.WorkflowResponse{
		Workflow: types.Workflow{
			ID:        "wf-1",
			ProjectID: "proj-test",
			Name:      "Pipeline",
			Slug:      "pipeline",
			CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		},
		Steps: []types.WorkflowStep{
			{ID: "step-1", WorkflowID: "wf-1", StepRef: "extract"},
			{ID: "step-2", WorkflowID: "wf-1", StepRef: "load", DependsOn: []string{"extract"}},
		},
	}
}

func TestWorkflowsClone_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"POST /v1/workflows/pipeline/clone": func(w http.ResponseWriter, _ *http.Request) {
			cloned := wf
			cloned.Slug = "pipeline-clone"
			respondJSON(t, w, http.StatusCreated, cloned)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsCloneCommand(state)
	cmd.SetArgs([]string{"pipeline", "--slug", "pipeline-clone"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "pipeline-clone") {
		t.Fatalf("expected cloned slug: %s", out)
	}
}

func TestWorkflowsDryRun_RejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowsDryRunCommand(state)
	cmd.SetArgs([]string{"pipeline", "--payload", "not-json"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected invalid JSON error, got: %v", err)
	}
}

func TestWorkflowsDryRun_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"POST /v1/workflows/pipeline/dry-run": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"ok":true}`))
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsDryRunCommand(state)
	cmd.SetArgs([]string{"pipeline", "--payload", `{"x":1}`})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsPlan_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"POST /v1/workflows/pipeline/plan": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"steps":[]}`))
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsPlanCommand(state)
	cmd.SetArgs([]string{"pipeline"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsSimulate_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"POST /v1/workflows/pipeline/simulate": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"simulated":true}`))
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsSimulateCommand(state)
	cmd.SetArgs([]string{"pipeline"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsVersions_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	versions := []types.WorkflowVersion{
		{WorkflowID: "wf-1", Version: 1, CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{WorkflowID: "wf-1", Version: 2, CreatedAt: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"GET /v1/workflows/pipeline/versions": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, versions)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsVersionsCommand(state)
	cmd.SetArgs([]string{"pipeline"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected versions in output: %s", out)
	}
}

func TestWorkflowsDiff_RejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowsDiffCommand(state)
	cmd.SetArgs([]string{"pipeline", "--from", "abc", "--to", "2"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--from must be a positive integer") {
		t.Fatalf("expected --from error, got: %v", err)
	}
}

func TestWorkflowsDiff_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	diff := types.WorkflowDiff{WorkflowID: "wf-1", From: 1, To: 2, Changes: json.RawMessage(`{}`)}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"GET /v1/workflows/pipeline/diff": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "from", "1")
			assertQuery(t, r, "to", "2")
			respondJSON(t, w, http.StatusOK, diff)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsDiffCommand(state)
	cmd.SetArgs([]string{"pipeline", "--from", "1", "--to", "2"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsPolicy_Get(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	policy := types.WorkflowPolicy{
		WorkflowID: "wf-1",
		Policy:     json.RawMessage(`{"max_concurrency":1}`),
		UpdatedAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"GET /v1/workflows/pipeline/policy": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, policy)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsPolicyCommand(state)
	cmd.SetArgs([]string{"pipeline"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "max_concurrency") {
		t.Fatalf("expected policy in output: %s", out)
	}
}

func TestWorkflowsPolicy_Set(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	policy := types.WorkflowPolicy{WorkflowID: "wf-1", Policy: json.RawMessage(`{"max_concurrency":2}`)}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"PUT /v1/workflows/pipeline/policy": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, policy)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsPolicyCommand(state)
	cmd.SetArgs([]string{"pipeline", "--set", `{"max_concurrency":2}`})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsVisualize_TableOutput(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "table"
	cmd := newWorkflowsVisualizeCommand(state)
	cmd.SetArgs([]string{"pipeline"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "extract") || !strings.Contains(out, "load") {
		t.Fatalf("expected rendered DAG in output: %s", out)
	}
}

func TestWorkflowsVisualize_JSONIncludesRawDAGData(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForExtras()
	stepRuns := []types.WorkflowStepRun{{ID: "step-run-1", WorkflowRunID: "wfr-1", StepRef: "extract", Status: types.StepCompleted}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
		"GET /v1/workflow-runs/wfr-1/steps": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, stepRuns)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsVisualizeCommand(state)
	cmd.SetArgs([]string{"pipeline", "--run", "wfr-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"steps"`) || !strings.Contains(out, `"statuses"`) || !strings.Contains(out, "completed") {
		t.Fatalf("expected raw DAG data in JSON output: %s", out)
	}
}
