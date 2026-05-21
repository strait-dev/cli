package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

// withStubTunnel swaps devStartTunnel for a fake that immediately returns
// the given URL. The shutdown function records the call so tests can assert
// cleanup happened.
func withStubTunnel(t *testing.T, url string) *bool {
	t.Helper()
	stopped := false
	prev := devStartTunnel
	devStartTunnel = func(_ context.Context, _ int) (string, func() error, error) {
		return url, func() error {
			stopped = true
			return nil
		}, nil
	}
	t.Cleanup(func() { devStartTunnel = prev })
	return &stopped
}

func TestDev_RegistersAndRestoresEndpoints(t *testing.T) {
	stopped := withStubTunnel(t, "https://test.trycloudflare.com")

	config := ProjectConfig{
		Version: "1",
		Jobs: []ProjectJob{
			{Slug: "existing", EndpointURL: "ignored"},
			{Slug: "fresh", EndpointURL: "ignored"},
		},
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.json")
	data, _ := json.Marshal(config)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var updateLog []string
	creates := map[string]string{}
	listCalls := 0

	jobs := []types.Job{{ID: "job-existing", ProjectID: "proj-test", Slug: "existing", EndpointURL: "https://app.example.com/existing"}}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("project_id"); got != "proj-test" {
				http.Error(w, "missing project_id", http.StatusBadRequest)
				return
			}
			mu.Lock()
			listCalls++
			mu.Unlock()
			respondPaginated(t, w, http.StatusOK, jobs)
		},
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			var req client.CreateJobRequest
			readJSONBody(t, r, &req)
			mu.Lock()
			creates[req.Slug] = req.EndpointURL
			mu.Unlock()
			respondJSON(t, w, http.StatusCreated, types.Job{ID: "job-fresh", ProjectID: req.ProjectID, Slug: req.Slug, EndpointURL: req.EndpointURL})
		},
		"PATCH /v1/jobs/job-existing": func(w http.ResponseWriter, r *http.Request) {
			var req client.UpdateJobRequest
			readJSONBody(t, r, &req)
			mu.Lock()
			updateLog = append(updateLog, *req.EndpointURL)
			mu.Unlock()
			respondJSON(t, w, http.StatusOK, types.Job{ID: "job-existing", ProjectID: "proj-test", Slug: "existing", EndpointURL: *req.EndpointURL})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"

	cmd := newDevCommand(state)
	cmd.SetArgs([]string{"--file", configPath})

	// Run dev in a goroutine; deliver SIGINT once endpoints have been registered.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	// Wait until the patch lands then signal.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := len(updateLog) >= 1 && updateLog[0] == "https://test.trycloudflare.com/existing" && creates["fresh"] == "https://test.trycloudflare.com/fresh"
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send sigint: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("dev returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("dev did not exit after SIGINT")
	}

	if !*stopped {
		t.Fatal("expected tunnel stop function to be called")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(updateLog) == 0 || updateLog[0] != "https://test.trycloudflare.com/existing" {
		t.Fatalf("existing job not patched to tunnel URL: %v", updateLog)
	}
	if creates["fresh"] != "https://test.trycloudflare.com/fresh" {
		t.Fatalf("fresh job not created with tunnel URL: %v", creates)
	}
	if listCalls < 2 {
		t.Fatalf("expected restore phase to re-list jobs, got %d list calls", listCalls)
	}
	if len(updateLog) < 2 || updateLog[len(updateLog)-1] != "https://app.example.com/existing" {
		t.Fatalf("expected restore phase to revert endpoint, got updates=%v", updateLog)
	}
}

func TestDev_RejectsMissingProject(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, srv)
	state.opts.projectID = ""

	cmd := newDevCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || err.Error() == "" {
		t.Fatal("expected error for missing project")
	}
}

func TestDev_RejectsMissingProjectConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, srv)
	cmd := newDevCommand(state)
	cmd.SetArgs([]string{"--file", "/nonexistent/strait.json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestDevCommand_Wiring(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	dev := findSubcommand(t, cmd, "dev")
	for _, flag := range []string{"port", "dir", "file", "keep-endpoint"} {
		if dev.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
