package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

func writeProjectConfig(t *testing.T, m ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "strait.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestSync_PlansCreateUpdateSkip(t *testing.T) {
	t.Parallel()

	existing := []types.Job{
		{ID: "job-keep", ProjectID: "proj-test", Slug: "keep", EndpointURL: "https://app.example.com/keep"},
		{ID: "job-change", ProjectID: "proj-test", Slug: "change", EndpointURL: "https://app.example.com/old"},
	}
	config := ProjectConfig{
		Version: "1",
		Jobs: []ProjectJob{
			{Slug: "keep", EndpointURL: "https://app.example.com/keep"},
			{Slug: "change", EndpointURL: "https://app.example.com/new"},
			{Slug: "fresh", EndpointURL: "https://app.example.com/fresh"},
		},
	}
	configPath := writeProjectConfig(t, config)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, existing)
		},
	})
	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", configPath, "--dry-run"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var summary SyncSummary
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("parse summary: %v\n%s", err, out)
	}
	if summary.Created != 1 || summary.Updated != 1 || summary.Skipped != 1 {
		t.Fatalf("plan counts: %+v", summary)
	}
}

func TestSync_AppliesCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existing := []types.Job{
		{ID: "job-change", ProjectID: "proj-test", Slug: "change", EndpointURL: "https://app.example.com/old"},
	}
	config := ProjectConfig{
		Version: "1",
		Jobs: []ProjectJob{
			{Slug: "change", EndpointURL: "https://app.example.com/new"},
			{Slug: "fresh", EndpointURL: "https://app.example.com/fresh"},
		},
	}
	configPath := writeProjectConfig(t, config)

	var mu sync.Mutex
	created := 0
	updated := 0

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, existing)
		},
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			var req client.CreateJobRequest
			readJSONBody(t, r, &req)
			mu.Lock()
			created++
			mu.Unlock()
			respondJSON(t, w, http.StatusCreated, types.Job{ID: "job-fresh", ProjectID: req.ProjectID, Slug: req.Slug, EndpointURL: req.EndpointURL})
		},
		"PATCH /v1/jobs/job-change": func(w http.ResponseWriter, r *http.Request) {
			var req client.UpdateJobRequest
			readJSONBody(t, r, &req)
			mu.Lock()
			updated++
			mu.Unlock()
			respondJSON(t, w, http.StatusOK, types.Job{ID: "job-change", ProjectID: "proj-test", Slug: "change", EndpointURL: *req.EndpointURL})
		},
	})

	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", configPath})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mu.Lock()
	defer mu.Unlock()
	if created != 1 {
		t.Fatalf("expected 1 create, got %d", created)
	}
	if updated != 1 {
		t.Fatalf("expected 1 update, got %d", updated)
	}
	var summary SyncSummary
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("parse summary: %v\n%s", err, out)
	}
	if summary.Created != 1 || summary.Updated != 1 {
		t.Fatalf("expected created=1 updated=1, got %+v", summary)
	}
}

func TestSync_PruneRequiresYes(t *testing.T) {
	t.Parallel()

	config := ProjectConfig{Version: "1", Jobs: []ProjectJob{{Slug: "keep", EndpointURL: "https://x/y"}}}
	configPath := writeProjectConfig(t, config)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{})
		},
	})
	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", configPath, "--prune"})

	// CI mode (test state default) + prune without --yes should refuse.
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error requiring --yes for prune in CI mode")
	}
}

func TestSync_PruneDeletesExtras(t *testing.T) {
	t.Parallel()

	existing := []types.Job{
		{ID: "job-keep", ProjectID: "proj-test", Slug: "keep", EndpointURL: "https://x/keep"},
		{ID: "job-stale", ProjectID: "proj-test", Slug: "stale", EndpointURL: "https://x/stale"},
	}
	config := ProjectConfig{Version: "1", Jobs: []ProjectJob{{Slug: "keep", EndpointURL: "https://x/keep"}}}
	configPath := writeProjectConfig(t, config)

	var mu sync.Mutex
	var deleted []string

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, existing)
		},
		"DELETE /v1/jobs/job-stale": func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = append(deleted, "job-stale")
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		},
	})
	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", configPath, "--prune", "--yes"})

	if _, err := captureRun(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(deleted) != 1 || deleted[0] != "job-stale" {
		t.Fatalf("expected job-stale to be deleted, got %v", deleted)
	}
}

func TestSync_RejectsMissingProjectConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", "/nonexistent/strait.json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func TestSync_RejectsInvalidProjectConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "strait.json")
	if err := os.WriteFile(path, []byte(`{"jobs":[{"slug":"x"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, srv)
	cmd := newSyncCommand(state)
	cmd.SetArgs([]string{"--file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for config missing endpoint_url")
	}
	if !strings.Contains(err.Error(), "endpoint_url") {
		t.Fatalf("expected endpoint_url error, got: %v", err)
	}
}

func TestWorkflowStepRequests_ResolvesJobRef(t *testing.T) {
	t.Parallel()

	steps, err := workflowStepRequests([]ProjectWorkflowStep{
		{StepRef: "send", JobRef: "send-email", DependsOn: []string{"prepare"}},
	}, map[string]string{"send-email": "job-123"})
	if err != nil {
		t.Fatalf("workflowStepRequests: %v", err)
	}
	if len(steps) != 1 || steps[0].JobID != "job-123" || steps[0].StepRef != "send" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestWorkflowStepRequests_RejectsUnknownJobRef(t *testing.T) {
	t.Parallel()

	_, err := workflowStepRequests([]ProjectWorkflowStep{{StepRef: "send", JobRef: "missing"}}, nil)
	if err == nil {
		t.Fatal("expected unknown job_ref error")
	}
	if !strings.Contains(err.Error(), "unknown job_ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCommand_Wiring(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	syncCmd := findSubcommand(t, cmd, "sync")
	for _, flag := range []string{"dir", "file", "dry-run", "prune", "yes"} {
		if syncCmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

func TestDeployCommand_Wiring(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	deploy := findSubcommand(t, cmd, "deploy")
	push := findSubcommand(t, deploy, "push")
	for _, flag := range []string{"dir", "file", "manifest", "dry-run", "prune", "yes"} {
		if push.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

// captureRun is a thin convenience over captureStateOutput that returns both
// the captured output and the error from cmd.Execute. It avoids the nested
// closure when the test wants to assert on both.
func captureRun(t *testing.T, state *appState, cmd interface{ Execute() error }) (string, error) {
	t.Helper()
	var execErr error
	out := captureStateOutput(t, state, func() {
		execErr = cmd.Execute()
	})
	return out, execErr
}
