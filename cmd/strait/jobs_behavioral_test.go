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

var testJob = types.Job{
	ID:          "job-1",
	ProjectID:   "proj-test",
	Name:        "Test Job",
	Slug:        "test-job",
	EndpointURL: "https://example.com/hook",
	MaxAttempts: 3,
	TimeoutSecs: 60,
	Enabled:     true,
	Version:     1,
	CreatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

func TestJobsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job-1 in output, got: %s", out)
	}
}

func TestJobsList_NoProject(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newJobsListCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected project ID error, got: %v", err)
	}
}

func TestJobsGet_ByID(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsGetCommand(state)
	cmd.SetArgs([]string{"job-1"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job-1 in output, got: %s", out)
	}
}

func TestJobsGet_BySlug_Resolves(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/test-job": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "not found")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsGetCommand(state)
	cmd.SetArgs([]string{"test-job"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected resolved job-1 in output, got: %s", out)
	}
}

func TestJobsCreate_Success(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "Test Job",
		"--slug", "test-job",
		"--endpoint", "https://example.com/hook",
	})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if receivedBody["name"] != "Test Job" {
		t.Fatalf("expected name=Test Job in body, got: %v", receivedBody)
	}
	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job-1 in output, got: %s", out)
	}
}

func TestJobsCreate_MissingFields(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestJobsTrigger_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"POST /v1/jobs/job-1/trigger": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var body map[string]any
			readJSONBody(t, r, &body)
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id":     "run-1",
				"status": "queued",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsTriggerCommand(state)
	cmd.SetArgs([]string{"job-1", "--payload", `{"key":"value"}`})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestJobsTrigger_InvalidPayload(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsTriggerCommand(state)
	cmd.SetArgs([]string{"job-1", "--payload", "not-json"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "payload must be valid JSON") {
		t.Fatalf("expected invalid payload error, got: %v", err)
	}
}

func TestJobsDelete_WithYes(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"DELETE /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsDeleteCommand(state)
	cmd.SetArgs([]string{"job-1", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestJobsDelete_CIBlocksPrompt(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	state.opts.ciMode = true
	cmd := newJobsDeleteCommand(state)
	cmd.SetArgs([]string{"job-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("expected non-interactive error, got: %v", err)
	}
}

func TestJobsVersions_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"GET /v1/jobs/job-1/versions": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.JobVersion{
				{ID: "jv-1", JobID: "job-1", Version: 1, Name: "v1", Slug: "test-job", CreatedAt: now},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsVersionsCommand(state)
	cmd.SetArgs([]string{"job-1"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "jv-1") {
		t.Fatalf("expected jv-1 in output, got: %s", out)
	}
}

func TestJobsEdit_FieldUpdate(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &patchBody)
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsEditCommand(state)
	cmd.SetArgs([]string{"job-1", "--field", "name=Updated Name"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patchBody["name"] != "Updated Name" {
		t.Fatalf("expected name=Updated Name in PATCH body, got: %v", patchBody)
	}
}

func TestJobsTriggerBulk_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"POST /v1/jobs/job-1/trigger/bulk": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"results": []map[string]any{{"id": "run-1", "status": "queued"}},
				"total":   1,
				"created": 1,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsTriggerBulkCommand(state)
	items, _ := json.Marshal([]map[string]any{{"payload": map[string]any{"id": "1"}}})
	cmd.SetArgs([]string{"job-1", "--items-json", string(items)})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
}

func TestJobsTriggerBulk_EmptyItems(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsTriggerBulkCommand(state)
	cmd.SetArgs([]string{"job-1", "--items-json", "[]"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected empty items error, got: %v", err)
	}
}

func TestResolveJobIdentifier_ByID(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	id, err := resolveJobIdentifier(t.Context(), cli, state, "job-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != "job-1" {
		t.Fatalf("expected job-1, got %s", id)
	}
}

func TestResolveJobIdentifier_BySlug(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/test-job": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "not found")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	id, err := resolveJobIdentifier(t.Context(), cli, state, "test-job")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != "job-1" {
		t.Fatalf("expected job-1, got %s", id)
	}
}

func TestResolveJobIdentifier_NotFound(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/nope": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "not found")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{})
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	_, err = resolveJobIdentifier(t.Context(), cli, state, "nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestJobsList_ShowsSourceType(t *testing.T) {
	t.Parallel()

	codeJob := types.Job{
		ID:                 "job-code",
		ProjectID:          "proj-test",
		Name:               "Code Job",
		Slug:               "code-job",
		EndpointURL:        "",
		MaxAttempts:        3,
		TimeoutSecs:        60,
		Enabled:            true,
		Version:            3,
		SourceType:         "code",
		ActiveDeploymentID: "dep-abc",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	endpointJob := types.Job{
		ID:          "job-ep",
		ProjectID:   "proj-test",
		Name:        "Endpoint Job",
		Slug:        "endpoint-job",
		EndpointURL: "https://example.com/hook",
		MaxAttempts: 3,
		TimeoutSecs: 60,
		Enabled:     true,
		Version:     1,
		SourceType:  "endpoint",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{codeJob, endpointJob})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	var rows []map[string]any
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// First row should be code job.
	src, _ := rows[0]["source_type"].(string)
	if src != "code" {
		t.Fatalf("expected source_type=code, got %q", src)
	}
	dep, _ := rows[0]["active_deployment_id"].(string)
	if dep != "dep-abc" {
		t.Fatalf("expected active_deployment_id=dep-abc, got %q", dep)
	}

	// Second row should be endpoint job.
	src2, _ := rows[1]["source_type"].(string)
	if src2 != "endpoint" {
		t.Fatalf("expected source_type=endpoint, got %q", src2)
	}
}

func TestJobSourceDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"code", "code"},
		{"endpoint", "endpoint"},
		{"", "endpoint"},
		{"custom", "custom"},
	}
	for _, tc := range tests {
		got := jobSourceDisplay(tc.input)
		if got != tc.want {
			t.Errorf("jobSourceDisplay(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDeploymentWatch_ExitsImmediatelyIfReady(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-1", Slug: "my-job", ProjectID: "proj-test"},
			})
		},
		"GET /v1/jobs/job-1/deployments/dep-ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, client.CodeDeployment{
				ID: "dep-ready", Status: "ready", Version: 3, Runtime: "go",
				BuiltImageURI: "registry.io/app:abc123",
			})
		},
	})

	state := newTestState(t, srv)
	watchCmd, _ := newCodeDeploymentWatchCommand(state)
	watchCmd.SetArgs([]string{"dep-ready", "--job", "my-job"})

	if err := watchCmd.Execute(); err != nil {
		t.Fatalf("watch should exit 0 when deployment is ready, got: %v", err)
	}
}

func TestDeploymentWatch_ExitsOneWhenFailed(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-1", Slug: "my-job", ProjectID: "proj-test"},
			})
		},
		"GET /v1/jobs/job-1/deployments/dep-fail": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, client.CodeDeployment{
				ID: "dep-fail", Status: "failed", Version: 1, Runtime: "go",
				ErrorMessage: "compilation failed",
			})
		},
	})

	state := newTestState(t, srv)
	watchCmd, _ := newCodeDeploymentWatchCommand(state)
	watchCmd.SetArgs([]string{"dep-fail", "--job", "my-job"})

	err := watchCmd.Execute()
	if err == nil {
		t.Fatal("watch should exit non-zero when deployment has failed")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected 'failed' in error message, got: %v", err)
	}
}

func TestDeploymentWatch_RequiresJobFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	watchCmd, _ := newCodeDeploymentWatchCommand(state)
	watchCmd.SetArgs([]string{"dep-1"})

	err := watchCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--job") {
		t.Fatalf("expected --job required error, got: %v", err)
	}
}
