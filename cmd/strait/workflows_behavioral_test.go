package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testWorkflow = types.Workflow{
	ID:        "wf-1",
	ProjectID: "proj-test",
	Name:      "Test Workflow",
	Slug:      "test-workflow",
	Enabled:   true,
	Version:   1,
	CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

var testWorkflowStep = types.WorkflowStep{
	ID:         "ws-1",
	WorkflowID: "wf-1",
	JobID:      "job-1",
	StepRef:    "step-a",
	DependsOn:  []string{},
	OnFailure:  types.FailWorkflow,
	StepType:   types.WorkflowStepTypeJob,
	CreatedAt:  time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

// workflowResponseJSON builds the combined JSON object that the server returns
// for workflow get/create/update endpoints (WorkflowResponse = Workflow + Steps).
func workflowResponseJSON() map[string]any {
	return map[string]any{
		"id":         testWorkflow.ID,
		"project_id": testWorkflow.ProjectID,
		"name":       testWorkflow.Name,
		"slug":       testWorkflow.Slug,
		"enabled":    testWorkflow.Enabled,
		"version":    testWorkflow.Version,
		"created_at": testWorkflow.CreatedAt,
		"updated_at": testWorkflow.UpdatedAt,
		"steps": []map[string]any{
			{
				"id":          testWorkflowStep.ID,
				"workflow_id": testWorkflowStep.WorkflowID,
				"job_id":      testWorkflowStep.JobID,
				"step_ref":    testWorkflowStep.StepRef,
				"depends_on":  testWorkflowStep.DependsOn,
				"on_failure":  string(testWorkflowStep.OnFailure),
				"step_type":   string(testWorkflowStep.StepType),
				"created_at":  testWorkflowStep.CreatedAt,
			},
		},
	}
}

func TestWorkflowsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.Workflow{testWorkflow})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected wf-1 in output, got: %s", out)
	}
}

func TestWorkflowsGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsGetCommand(state)
	cmd.SetArgs([]string{"wf-1"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected wf-1 in output, got: %s", out)
	}
}

func TestWorkflowsCreate_Success(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflows": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, workflowResponseJSON())
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "Test Workflow",
		"--slug", "test-workflow",
		"--description", "A test workflow",
	})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if receivedBody["name"] != "Test Workflow" {
		t.Fatalf("expected name=Test Workflow in body, got: %v", receivedBody)
	}
	if receivedBody["slug"] != "test-workflow" {
		t.Fatalf("expected slug=test-workflow in body, got: %v", receivedBody)
	}
	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected wf-1 in output, got: %s", out)
	}
}

func TestWorkflowsCreate_MissingFields(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newWorkflowsCreateCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project, name, and slug are required") {
		t.Fatalf("expected missing fields error, got: %v", err)
	}
}

func TestWorkflowsUpdate_Success(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
		"PATCH /v1/workflows/wf-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &patchBody)
			resp := workflowResponseJSON()
			resp["name"] = "Updated Name"
			respondJSON(t, w, http.StatusOK, resp)
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsUpdateCommand(state)
	cmd.SetArgs([]string{"wf-1", "--name", "Updated Name"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if patchBody["name"] != "Updated Name" {
		t.Fatalf("expected name=Updated Name in PATCH body, got: %v", patchBody)
	}
	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected wf-1 in output, got: %s", out)
	}
}

func TestWorkflowsUpdate_NoFlags(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWorkflowsUpdateCommand(state)
	cmd.SetArgs([]string{"wf-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one update flag is required") {
		t.Fatalf("expected update flag error, got: %v", err)
	}
}

func TestWorkflowsDelete_WithYes(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
		"DELETE /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "deleted"})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsDeleteCommand(state)
	cmd.SetArgs([]string{"wf-1", "--yes"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !deleteCalled {
		t.Fatal("expected DELETE to be called")
	}
	if !strings.Contains(out, "wf-1") {
		t.Fatalf("expected wf-1 in output, got: %s", out)
	}
}

func TestWorkflowsTrigger_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
		"POST /v1/workflows/wf-1/trigger": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, types.WorkflowRun{
				ID:          "wfr-1",
				WorkflowID:  "wf-1",
				ProjectID:   "proj-test",
				Status:      types.WfStatusPending,
				TriggeredBy: "api",
				CreatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newWorkflowsTriggerCommand(state)
	cmd.SetArgs([]string{"wf-1", "--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "wfr-1") {
		t.Fatalf("expected wfr-1 in output, got: %s", out)
	}
}

func TestWorkflowsVisualizeCommand(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "" // exercise the rendered ASCII path
	cmd := newWorkflowsVisualizeCommand(state)
	cmd.SetArgs([]string{"wf-1"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "step-a") {
		t.Fatalf("expected step-a in visualize output, got: %s", out)
	}
}

func TestWorkflowsVisualizeCommand_FormatJSON(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, workflowResponseJSON())
		},
	})

	state := newTestState(t, srv) // outputFormat already "json"
	cmd := newWorkflowsVisualizeCommand(state)
	cmd.SetArgs([]string{"wf-1"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON DAG output, got %q (err=%v)", out, err)
	}
	if got["workflow_id"] != "wf-1" {
		t.Fatalf("expected workflow_id=wf-1, got: %v", got["workflow_id"])
	}
	nodes, ok := got["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		t.Fatalf("expected nodes in JSON output, got: %v", got)
	}
}
