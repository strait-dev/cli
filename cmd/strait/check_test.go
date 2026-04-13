package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckDAG_ErrorsOnNonArray(t *testing.T) {
	t.Parallel()

	err := checkDAG("not-an-array")
	if err == nil || !strings.Contains(err.Error(), "steps must be an array") {
		t.Fatalf("expected array error, got: %v", err)
	}
}

func TestCheckDAG_ErrorsOnDuplicateStepRef(t *testing.T) {
	t.Parallel()

	err := checkDAG([]any{
		map[string]any{"step_ref": "a"},
		map[string]any{"step_ref": "a"},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate step_ref "a"`) {
		t.Fatalf("expected duplicate step_ref error, got: %v", err)
	}
}

func TestCheckDAG_ErrorsOnUnknownDependency(t *testing.T) {
	t.Parallel()

	err := checkDAG([]any{
		map[string]any{"step_ref": "a", "depends_on": []any{"missing"}},
	})
	if err == nil || !strings.Contains(err.Error(), `depends on unknown step "missing"`) {
		t.Fatalf("expected unknown dependency error, got: %v", err)
	}
}

func TestCheckDAG_ErrorsOnCycle(t *testing.T) {
	t.Parallel()

	err := checkDAG([]any{
		map[string]any{"step_ref": "a", "depends_on": []any{"b"}},
		map[string]any{"step_ref": "b", "depends_on": []any{"a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got: %v", err)
	}
}

func TestCheckDAG_AllowsAcyclicDependencies(t *testing.T) {
	t.Parallel()

	err := checkDAG([]any{
		map[string]any{"step_ref": "fetch"},
		map[string]any{"step_ref": "process", "depends_on": []any{"fetch"}},
		map[string]any{"step_ref": "notify", "depends_on": []any{"process"}},
	})
	if err != nil {
		t.Fatalf("expected acyclic DAG to pass, got: %v", err)
	}
}

func TestCheckEndpointReachable_InvalidURL(t *testing.T) {
	t.Parallel()

	err := checkEndpointReachable("://bad-url", time.Second)
	if err == nil || !strings.Contains(err.Error(), "invalid URL") {
		t.Fatalf("expected invalid URL error, got: %v", err)
	}
}

func TestCheckCommand_WorkflowCycleFailsValidation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "workflow.yaml")
	content := `apiVersion: v1
kind: Workflow
metadata:
  name: cyclical
spec:
  project_id: proj-1
  steps:
    - step_ref: step-a
      depends_on: [step-b]
    - step_ref: step-b
      depends_on: [step-a]
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newCheckCommand(state)
	cmd.SetArgs([]string{"-f", manifestPath})

	out := captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "check failed") {
			t.Fatalf("expected check failed error, got: %v", err)
		}
	})

	if !strings.Contains(out, `"check": "dag"`) || !strings.Contains(out, `"ok": false`) {
		t.Fatalf("expected DAG failure in output, got:\n%s", out)
	}
}

func TestCheckCommand_InvalidCronFailsValidation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "job.yaml")
	content := `apiVersion: v1
kind: Job
metadata:
  name: invalid-cron
spec:
  project_id: proj-1
  endpoint_url: https://example.com/jobs/invalid-cron
  cron: bad cron expr
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newCheckCommand(state)
	cmd.SetArgs([]string{"-f", manifestPath})

	out := captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "check failed") {
			t.Fatalf("expected check failed error, got: %v", err)
		}
	})

	if !strings.Contains(out, `"check": "cron"`) || !strings.Contains(out, `"ok": false`) {
		t.Fatalf("expected cron failure in output, got:\n%s", out)
	}
}

func TestCheckCommand_EndpointReachabilitySuccess(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"HEAD /job": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "job.yaml")
	content := "apiVersion: v1\nkind: Job\nmetadata:\n  name: endpoint-check\nspec:\n  project_id: proj-1\n  endpoint_url: " + srv.URL + "/job\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newCheckCommand(state)
	cmd.SetArgs([]string{"-f", manifestPath, "--check-endpoints", "--endpoint-timeout", "2s"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"check": "endpoint"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("expected successful endpoint check in output, got:\n%s", out)
	}
}

func TestCheckCommand_EndpointReachabilityFailureReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "job.yaml")
	content := "apiVersion: v1\nkind: Job\nmetadata:\n  name: endpoint-check\nspec:\n  project_id: proj-1\n  endpoint_url: http://127.0.0.1:1/job\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newCheckCommand(state)
	cmd.SetArgs([]string{"-f", manifestPath, "--check-endpoints", "--endpoint-timeout", "200ms"})

	out := captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "check failed") {
			t.Fatalf("expected check failed error, got: %v", err)
		}
	})

	if !strings.Contains(out, `"check": "endpoint"`) || !strings.Contains(out, `"ok": false`) {
		t.Fatalf("expected failed endpoint check in output, got:\n%s", out)
	}
}
