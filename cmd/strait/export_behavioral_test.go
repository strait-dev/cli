package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func TestExportJobs_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-1", ProjectID: "proj-test", Name: "My Job", Slug: "my-job",
					EndpointURL: "https://example.com/hook", MaxAttempts: 3, TimeoutSecs: 60,
					Enabled: true, CreatedAt: now, UpdatedAt: now},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportCommand(state)
	cmd.SetArgs([]string{"jobs", "--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "My Job") {
		t.Fatalf("expected job name in YAML output, got: %s", out)
	}
	if !strings.Contains(out, "kind: Job") {
		t.Fatalf("expected 'kind: Job' in output, got: %s", out)
	}
}

func TestExportWorkflows_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	wf := types.Workflow{
		ID: "wf-1", ProjectID: "proj-test", Name: "Pipeline", Slug: "pipeline",
		Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now,
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Workflow{wf})
		},
		"GET /v1/workflows/wf-1": func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]any{
				"id": wf.ID, "project_id": wf.ProjectID, "name": wf.Name,
				"slug": wf.Slug, "enabled": wf.Enabled, "version": wf.Version,
				"created_at": now, "updated_at": now,
				"steps": []map[string]any{
					{"id": "step-1", "workflow_id": "wf-1", "step_ref": "send", "depends_on": []string{}, "on_failure": "fail_workflow", "created_at": now},
				},
			}
			respondJSON(t, w, http.StatusOK, resp)
		},
	})

	state := newTestState(t, srv)
	cmd := newExportCommand(state)
	cmd.SetArgs([]string{"workflows", "--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Pipeline") {
		t.Fatalf("expected workflow name in output, got: %s", out)
	}
}

func TestExportAll_DryRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-1", Name: "J1", Slug: "j1", EndpointURL: "https://x.com", CreatedAt: now, UpdatedAt: now},
			})
		},
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Workflow{})
		},
		"GET /v1/api-keys": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.APIKey{})
		},
	})

	state := newTestState(t, srv)
	cmd := newExportCommand(state)
	cmd.SetArgs([]string{"all", "--project", "proj-test", "--dry-run"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected JSON dry-run output, got: %s", out)
	}
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got: %v", result)
	}
}

func TestExport_UnsupportedResource(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newExportCommand(state)
	cmd.SetArgs([]string{"secrets", "--project", "proj-test"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported resource") {
		t.Fatalf("expected unsupported resource error, got: %v", err)
	}
}

func TestExport_NoProject(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newExportCommand(state)
	cmd.SetArgs([]string{"jobs"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected project ID error, got: %v", err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"My Job", "my-job"},
		{"path/to/thing", "path-to-thing"},
		{"back\\slash", "back-slash"},
		{"", ""},
		{"  SPACES  ", "spaces"},
		{"already-clean", "already-clean"},
	}

	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
