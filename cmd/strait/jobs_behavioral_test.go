package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	out := captureStateOutput(t, state, func() {
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

	out := captureStateOutput(t, state, func() {
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

	out := captureStateOutput(t, state, func() {
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

	out := captureStateOutput(t, state, func() {
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

	out := captureStateOutput(t, state, func() {
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

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !deleteCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestJobsDelete_TTYSuccessMessage(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"DELETE /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsDeleteCommand(state)
		cmd.SetArgs([]string{"job-1", "--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "Deleted job") || !strings.Contains(stderr, "job-1") {
		t.Fatalf("expected tty delete message, got: %s", stderr)
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

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "jv-1") {
		t.Fatalf("expected jv-1 in output, got: %s", out)
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
	out := captureStateOutput(t, state, func() {
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

func TestJobsList_TTYShowsCronFallback(t *testing.T) {
	t.Parallel()

	job := testJob
	job.Cron = ""
	job.SourceType = "code"

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{job})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsListCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Jobs", "test-job", "cron=--", "source=code"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in tty output, got: %s", want, stderr)
		}
	}
}

func TestJobsCreate_UsesStateProjectAndTTYMessage(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, testJob)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	state.opts.projectID = "proj-from-state"
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsCreateCommand(state)
		cmd.SetArgs([]string{"--name", "Test Job", "--slug", "test-job", "--endpoint", "https://example.com/hook"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if receivedBody["project_id"] != "proj-from-state" {
		t.Fatalf("expected project_id from state, got: %#v", receivedBody)
	}
	if !strings.Contains(stderr, "Created job") || !strings.Contains(stderr, "ID") || !strings.Contains(stderr, "job-1") {
		t.Fatalf("expected tty create message, got: %s", stderr)
	}
}

func TestJobsTrigger_ValidatesScheduledAtAndPayloadFile(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	t.Run("invalid scheduled-at", func(t *testing.T) {
		state := newTestState(t, srv)
		cmd := newJobsTriggerCommand(state)
		cmd.SetArgs([]string{"job-1", "--scheduled-at", "tomorrow"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid scheduled-at") {
			t.Fatalf("expected invalid scheduled-at error, got: %v", err)
		}
	})

	t.Run("missing payload file", func(t *testing.T) {
		state := newTestState(t, srv)
		cmd := newJobsTriggerCommand(state)
		cmd.SetArgs([]string{"job-1", "--payload-file", filepath.Join(t.TempDir(), "missing.json")})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "no such file") {
			t.Fatalf("expected payload file read error, got: %v", err)
		}
	})
}

func TestJobsTrigger_TTYUsesPayloadFileAndScheduledAt(t *testing.T) {
	t.Parallel()

	payloadPath := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(payloadPath, []byte(`{"source":"file"}`), 0o600); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	var triggerBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"POST /v1/jobs/job-1/trigger": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &triggerBody)
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "run-tty", "status": "queued"})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsTriggerCommand(state)
		cmd.SetArgs([]string{"job-1", "--payload-file", payloadPath, "--scheduled-at", "2026-03-20T10:00:00Z"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if triggerBody["scheduled_at"] != "2026-03-20T10:00:00Z" {
		t.Fatalf("expected scheduled_at in request, got: %#v", triggerBody)
	}
	payloadMap, ok := triggerBody["payload"].(map[string]any)
	if !ok || payloadMap["source"] != "file" {
		t.Fatalf("expected payload from file, got: %#v", triggerBody["payload"])
	}
	if !strings.Contains(stderr, "Triggered run") || !strings.Contains(stderr, "run-tty") {
		t.Fatalf("expected tty trigger message, got: %s", stderr)
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
