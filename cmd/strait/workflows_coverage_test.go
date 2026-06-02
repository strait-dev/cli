package main

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

// testWorkflowForCoverage returns a canonical workflow fixture for coverage tests.
func testWorkflowForCoverage() client.WorkflowResponse {
	return client.WorkflowResponse{
		Workflow: types.Workflow{
			ID:        "wf-cov-1",
			ProjectID: "proj-test",
			Name:      "Coverage Pipeline",
			Slug:      "cov-pipeline",
			CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		},
		Steps: []types.WorkflowStep{
			{ID: "step-1", WorkflowID: "wf-cov-1", StepRef: "ingest"},
			{ID: "step-2", WorkflowID: "wf-cov-1", StepRef: "process", DependsOn: []string{"ingest"}},
		},
	}
}

// workflowResolutionRouteT returns the GET /v1/workflows/<slug> route that
// resolveWorkflowIdentifier calls when the arg is a slug. All tests that
// pass a slug as the workflow argument must include this route.
func workflowResolutionRouteT(t *testing.T, wf client.WorkflowResponse) map[string]http.HandlerFunc {
	t.Helper()
	return map[string]http.HandlerFunc{
		"GET /v1/workflows/cov-pipeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, wf)
		},
	}
}

func mergeRoutes(a, b map[string]http.HandlerFunc) map[string]http.HandlerFunc {
	out := make(map[string]http.HandlerFunc, len(a)+len(b))
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}

func TestWorkflowsGraph_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForCoverage()
	routes := mergeRoutes(
		workflowResolutionRouteT(t, wf),
		map[string]http.HandlerFunc{
			"GET /v1/workflows/cov-pipeline/graph": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, json.RawMessage(`{"nodes":["ingest","process"]}`))
			},
		},
	)
	srv := newRouterServer(t, routes)
	state := newTestState(t, srv)
	cmd := newWorkflowsGraphCommand(state)
	cmd.SetArgs([]string{"cov-pipeline"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "nodes") {
		t.Fatalf("expected graph nodes in output: %s", out)
	}
}

func TestWorkflowsActiveVersions_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForCoverage()
	routes := mergeRoutes(
		workflowResolutionRouteT(t, wf),
		map[string]http.HandlerFunc{
			"GET /v1/workflows/cov-pipeline/active-versions": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, json.RawMessage(`{"active":["v1","v2"]}`))
			},
		},
	)
	srv := newRouterServer(t, routes)
	state := newTestState(t, srv)
	cmd := newWorkflowsActiveVersionsCommand(state)
	cmd.SetArgs([]string{"cov-pipeline"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsCanarySet_SendsCorrectBody(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForCoverage()
	var gotBody map[string]any

	routes := mergeRoutes(
		workflowResolutionRouteT(t, wf),
		map[string]http.HandlerFunc{
			"PATCH /v1/workflows/cov-pipeline/canary": func(w http.ResponseWriter, r *http.Request) {
				assertMethod(t, r, w, http.MethodPatch)
				readJSONBody(t, r, &gotBody)
				respondJSON(t, w, http.StatusOK, json.RawMessage(`{"traffic_pct":25}`))
			},
		},
	)
	srv := newRouterServer(t, routes)
	state := newTestState(t, srv)
	cmd := newWorkflowsCanarySetCommand(state)
	cmd.SetArgs([]string{"cov-pipeline", "--traffic-pct", "25"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pct, ok := gotBody["traffic_pct"]; !ok {
		t.Fatal("expected traffic_pct in request body")
	} else if pct.(float64) != 25 {
		t.Fatalf("expected traffic_pct=25, got %v", pct)
	}
}

func TestWorkflowsCanarySet_RequiresTrafficPct(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowsCanarySetCommand(state)
	cmd.SetArgs([]string{"cov-pipeline"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --traffic-pct")
	}
}

func TestWorkflowsCanaryGet_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForCoverage()
	routes := mergeRoutes(
		workflowResolutionRouteT(t, wf),
		map[string]http.HandlerFunc{
			"GET /v1/workflows/cov-pipeline/canary": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, json.RawMessage(`{"traffic_pct":10}`))
			},
		},
	)
	srv := newRouterServer(t, routes)
	state := newTestState(t, srv)
	cmd := newWorkflowsCanaryGetCommand(state)
	cmd.SetArgs([]string{"cov-pipeline"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "traffic_pct") {
		t.Fatalf("expected traffic_pct in output: %s", out)
	}
}

func TestWorkflowsCanaryRollback_Success(t *testing.T) {
	t.Parallel()

	wf := testWorkflowForCoverage()
	routes := mergeRoutes(
		workflowResolutionRouteT(t, wf),
		map[string]http.HandlerFunc{
			"POST /v1/workflows/cov-pipeline/canary/rollback": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, json.RawMessage(`{"rolled_back":true}`))
			},
		},
	)
	srv := newRouterServer(t, routes)
	state := newTestState(t, srv)
	cmd := newWorkflowsCanaryRollbackCommand(state)
	cmd.SetArgs([]string{"cov-pipeline"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsTimeline_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfrun-cov-1/timeline": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"events":[{"type":"started"},{"type":"completed"}]}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsTimelineCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "events") {
		t.Fatalf("expected timeline events in output: %s", out)
	}
}

func TestWorkflowRunsTimeline_RejectsInvalidID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsTimelineCommand(state)
	cmd.SetArgs([]string{"BAD ID WITH SPACES"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid workflow run id") {
		t.Fatalf("expected invalid id error, got: %v", err)
	}
}

func TestWorkflowRunsBulkCancel_SendsCorrectBody(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/bulk-cancel": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodPost)
			readJSONBody(t, r, &gotBody)
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"cancelled":2}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsBulkCancelCommand(state)
	cmd.SetArgs([]string{"--id", "wfrun-1", "--id", "wfrun-2"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, ok := gotBody["workflow_run_ids"]
	if !ok {
		t.Fatal("expected workflow_run_ids in request body")
	}
	idList, ok := ids.([]any)
	if !ok || len(idList) != 2 {
		t.Fatalf("expected 2 workflow_run_ids, got: %v", ids)
	}
	if idList[0].(string) != "wfrun-1" || idList[1].(string) != "wfrun-2" {
		t.Fatalf("unexpected workflow_run_ids: %v", idList)
	}
}

func TestWorkflowRunsBulkCancel_RequiresAtLeastOneID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsBulkCancelCommand(state)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --id")
	}
}

func TestWorkflowRunsBulkReplay_SendsCorrectBody(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/bulk-replay": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, w, http.MethodPost)
			readJSONBody(t, r, &gotBody)
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"replayed":1}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsBulkReplayCommand(state)
	cmd.SetArgs([]string{"--id", "wfrun-3"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, ok := gotBody["workflow_run_ids"]
	if !ok {
		t.Fatal("expected workflow_run_ids in request body")
	}
	idList, ok := ids.([]any)
	if !ok || len(idList) != 1 {
		t.Fatalf("expected 1 workflow_run_id, got: %v", ids)
	}
	if idList[0].(string) != "wfrun-3" {
		t.Fatalf("unexpected workflow_run_id: %v", idList[0])
	}
}

func TestWorkflowRunsReplaySubtree_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-cov-1/steps/ingest/replay-subtree": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"replayed":true}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsReplaySubtreeCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1", "--step", "ingest"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsReplaySubtree_RequiresStep(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsReplaySubtreeCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --step")
	}
}

func TestWorkflowRunsReplaySubtree_RejectsInvalidStepRef(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowRunsReplaySubtreeCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1", "--step", "bad step ref"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid step ref") {
		t.Fatalf("expected step ref error, got: %v", err)
	}
}

func TestWorkflowRunsCompare_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflow-runs/wfrun-cov-1/compare/wfrun-cov-2": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"diff":"none"}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsCompareCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1", "--other", "wfrun-cov-2"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowRunsCompensate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflow-runs/wfrun-cov-1/compensate": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, json.RawMessage(`{"compensated":true}`))
		},
	})
	state := newTestState(t, srv)
	cmd := newWorkflowRunsCompensateCommand(state)
	cmd.SetArgs([]string{"wfrun-cov-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterWorkflowsCoverageCommands_AddsSubcommands(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	wfCmd := newWorkflowsCommand(state)
	registerWorkflowsCoverageCommands(wfCmd, state)

	expected := []string{
		"graph", "active-versions", "version-get", "version-impact",
		"version-steps", "canary-get", "canary-set", "canary-rollback",
	}
	assertSubcommands(t, wfCmd, expected)
}

func TestRegisterWorkflowRunsCoverageCommands_AddsSubcommands(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	wfrCmd := newWorkflowRunsCommand(state)
	registerWorkflowRunsCoverageCommands(wfrCmd, state)

	expected := []string{
		"compare", "compensation-plan", "debug", "explain", "graph",
		"labels", "timeline", "compensate", "replay-subtree",
		"bulk-cancel", "bulk-replay",
	}
	assertSubcommands(t, wfrCmd, expected)
}
