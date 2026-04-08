package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/types"
)

func TestGenerateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"My Job Name", "my-job-name"},
		{"  spaces  ", "spaces"},
		{"UPPER-CASE", "upper-case"},
		{"special!@#chars", "specialchars"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"trailing-", "trailing"},
		{"123-numbers", "123-numbers"},
		{"", ""},
		{"already-slug", "already-slug"},
	}

	for _, tc := range tests {
		t.Run("input="+tc.input, func(t *testing.T) {
			t.Parallel()
			got := generateSlug(tc.input)
			if got != tc.want {
				t.Fatalf("generateSlug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCreateJob_JSONModeWithProjectInBody(t *testing.T) {
	t.Parallel()

	// JSON has project_id, no --project flag, no project in state
	state := &appState{opts: &rootOptions{}}
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{"--json"})

	input := `{"project_id":"proj-from-json","name":"test","slug":"test","endpoint_url":"http://example.com"}`
	cmd.SetIn(bytes.NewBufferString(input))

	err := cmd.Execute()
	// Should NOT fail with "project ID is required" since JSON body has it.
	// It will fail with a network error trying to reach the API, which is expected.
	if err != nil && strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("should not require --project when JSON body has project_id: %v", err)
	}
}

func TestCreateJob_JSONModeWithoutProject(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{"--json"})

	input := `{"name":"test","slug":"test","endpoint_url":"http://example.com"}`
	cmd.SetIn(bytes.NewBufferString(input))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected 'project ID is required' error, got: %v", err)
	}
}

func TestCreateJob_JSONModeFallsBackToFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{projectID: "proj-from-flag"}}
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{"--json"})

	input := `{"name":"test","slug":"test","endpoint_url":"http://example.com"}`
	cmd.SetIn(bytes.NewBufferString(input))

	err := cmd.Execute()
	// Should NOT fail with "project ID is required" since state has project.
	// It will fail with a network error, which is expected.
	if err != nil && strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("should use project from state when JSON body lacks it: %v", err)
	}
}

func TestCreateWorkflow_JSONModeWithProjectInBody(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newCreateWorkflowCommand(state)
	cmd.SetArgs([]string{"--json"})

	input := `{"project_id":"proj-from-json","name":"test","slug":"test"}`
	cmd.SetIn(bytes.NewBufferString(input))

	err := cmd.Execute()
	if err != nil && strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("should not require --project when JSON body has project_id: %v", err)
	}
}

func TestCreateWorkflow_JSONModeWithoutProject(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newCreateWorkflowCommand(state)
	cmd.SetArgs([]string{"--json"})

	input := `{"name":"test","slug":"test"}`
	cmd.SetIn(bytes.NewBufferString(input))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected 'project ID is required' error, got: %v", err)
	}
}

func TestCreateJob_IdempotencyKeyIsSentAsHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	job := types.Job{ID: "job-1", Slug: "my-job", Name: "My Job"}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			capturedKey = r.Header.Get("X-Idempotency-Key")
			respondJSON(t, w, http.StatusCreated, job)
		},
	})

	state := newTestState(t, srv)
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-1",
		"--name", "My Job",
		"--slug", "my-job",
		"--endpoint", "http://localhost:3000/jobs",
		"--idempotency-key", "key-abc-123",
	})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedKey != "key-abc-123" {
		t.Errorf("X-Idempotency-Key header: got %q, want %q", capturedKey, "key-abc-123")
	}
}

func TestCreateJob_NoIdempotencyKeyOmitsHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	job := types.Job{ID: "job-1", Slug: "my-job", Name: "My Job"}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			capturedKey = r.Header.Get("X-Idempotency-Key")
			respondJSON(t, w, http.StatusCreated, job)
		},
	})

	state := newTestState(t, srv)
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-1",
		"--name", "My Job",
		"--slug", "my-job",
		"--endpoint", "http://localhost:3000/jobs",
	})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedKey != "" {
		t.Errorf("expected empty X-Idempotency-Key when not provided, got %q", capturedKey)
	}
}

func TestCreateWorkflow_IdempotencyKeyIsSentAsHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/workflows": func(w http.ResponseWriter, r *http.Request) {
			capturedKey = r.Header.Get("X-Idempotency-Key")
			respondJSON(t, w, http.StatusCreated, map[string]any{
				"id": "wf-1", "slug": "my-workflow", "name": "My Workflow",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newCreateWorkflowCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-1",
		"--name", "My Workflow",
		"--slug", "my-workflow",
		"--idempotency-key", "wf-key-xyz",
	})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedKey != "wf-key-xyz" {
		t.Errorf("X-Idempotency-Key header: got %q, want %q", capturedKey, "wf-key-xyz")
	}
}

func TestCreateJob_HasIdempotencyKeyFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newCreateJobCommand(state)
	if cmd.Flags().Lookup("idempotency-key") == nil {
		t.Error("expected --idempotency-key flag on create job command")
	}
}

func TestCreateWorkflow_HasIdempotencyKeyFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newCreateWorkflowCommand(state)
	if cmd.Flags().Lookup("idempotency-key") == nil {
		t.Error("expected --idempotency-key flag on create workflow command")
	}
}

func TestJobsCreate_HasIdempotencyKeyFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newJobsCreateCommand(state)
	if cmd.Flags().Lookup("idempotency-key") == nil {
		t.Error("expected --idempotency-key flag on jobs create command")
	}
}

func TestCreateJob_JSONModeForwardsIdempotencyKey(t *testing.T) {
	t.Parallel()

	var capturedKey string
	job := types.Job{ID: "job-1", Slug: "my-job", Name: "My Job"}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			capturedKey = r.Header.Get("X-Idempotency-Key")
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			respondJSON(t, w, http.StatusCreated, job)
		},
	})

	state := newTestState(t, srv)
	cmd := newCreateJobCommand(state)
	cmd.SetArgs([]string{
		"--json",
		"--project", "proj-1",
		"--idempotency-key", "json-mode-key",
	})

	input := `{"name":"my-job","slug":"my-job","endpoint_url":"http://localhost:3000"}`
	cmd.SetIn(bytes.NewBufferString(input))

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedKey != "json-mode-key" {
		t.Errorf("X-Idempotency-Key in JSON mode: got %q, want %q", capturedKey, "json-mode-key")
	}
}
