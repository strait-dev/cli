package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

func testJobForExtras() types.Job {
	return types.Job{
		ID:        "job-1",
		ProjectID: "proj-test",
		Name:      "Backfill",
		Slug:      "backfill",
		Enabled:   true,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestJobsClone_Success(t *testing.T) {
	t.Parallel()

	job := testJobForExtras()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/backfill": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
		"POST /v1/jobs/backfill/clone": func(w http.ResponseWriter, _ *http.Request) {
			cloned := job
			cloned.ID = "job-2"
			cloned.Slug = "backfill-clone"
			respondJSON(t, w, http.StatusCreated, cloned)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsCloneCommand(state)
	cmd.SetArgs([]string{"backfill", "--slug", "backfill-clone"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "backfill-clone") {
		t.Fatalf("expected cloned slug in output: %s", out)
	}
}

func TestJobsHealth_Success(t *testing.T) {
	t.Parallel()

	job := testJobForExtras()
	health := types.JobHealth{
		JobID:         "job-1",
		Status:        "healthy",
		LastRunStatus: "succeeded",
		SuccessRate:   0.98,
		P95DurationMS: 1200,
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/backfill": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
		"GET /v1/jobs/backfill/health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, health)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsHealthCommand(state)
	cmd.SetArgs([]string{"backfill"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "healthy") {
		t.Fatalf("expected status in output: %s", out)
	}
}

func TestJobsDependencies_Success(t *testing.T) {
	t.Parallel()

	job := testJobForExtras()
	deps := []types.JobDependency{
		{ID: "dep-1", JobID: "job-1", DependsOn: "job-0", Type: "success"},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/backfill": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
		"GET /v1/jobs/backfill/dependencies": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, deps)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsDependenciesCommand(state)
	cmd.SetArgs([]string{"backfill"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "dep-1") {
		t.Fatalf("expected dep id in output: %s", out)
	}
}

func TestJobsAddDependency_RequiresDependsOn(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsAddDependencyCommand(state)
	cmd.SetArgs([]string{"backfill"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--depends-on is required") {
		t.Fatalf("expected --depends-on error, got: %v", err)
	}
}

func TestJobsAddDependency_Success(t *testing.T) {
	t.Parallel()

	job := testJobForExtras()
	dep := types.JobDependency{ID: "dep-1", JobID: "job-1", DependsOn: "job-0", Type: "success"}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/backfill": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
		"POST /v1/jobs/backfill/dependencies": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusCreated, dep)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsAddDependencyCommand(state)
	cmd.SetArgs([]string{"backfill", "--depends-on", "job-0", "--type", "success"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobsBatch_RequiresFromFile(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsBatchCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--from-file is required") {
		t.Fatalf("expected --from-file error, got: %v", err)
	}
}

func TestJobsBatch_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bad := filepath.Join(dir, "batch.json")
	if err := os.WriteFile(bad, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsBatchCommand(state)
	cmd.SetArgs([]string{"--from-file", bad})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid batch JSON") {
		t.Fatalf("expected JSON parse error, got: %v", err)
	}
}

func TestJobsBatch_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	good := filepath.Join(dir, "batch.json")
	body := `{"updates":[{"id":"job-1","patch":{"name":"Renamed"}}]}`
	if err := os.WriteFile(good, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	resp := client.BatchUpdateJobsResponse{Updated: []string{"job-1"}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs/batch": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, resp)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsBatchCommand(state)
	cmd.SetArgs([]string{"--from-file", good})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
