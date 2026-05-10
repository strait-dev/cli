package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDoctorCommand_FormatJSONOutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
		"GET /v1/system/capabilities": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"code_deploy_enabled": true})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureStateOutput(t, state, func() {
		_ = cmd.Execute()
	})

	var checks []doctorCheck
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		t.Fatalf("--format json output is not valid JSON array: %v\noutput: %s", err, out)
	}
	if len(checks) == 0 {
		t.Fatal("expected non-empty checks list")
	}
}

func TestDoctorCommand_NoJSONFlag(t *testing.T) {
	t.Parallel()

	// Verify the old --json flag is gone and produces an error.
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{"--json"})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for unknown --json flag, got nil")
		}
	})
}

func TestDoctorCheckEndpoints_StandaloneIncludesLiveJobCheck(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
		"GET /v1/system/capabilities": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"code_deploy_enabled": true})
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "job-1", "slug": "my-job", "name": "My Job", "endpoint_url": "http://localhost:3000/jobs/my-job", "enabled": true},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{"--check-endpoints"})

	out := captureStateOutput(t, state, func() {
		_ = cmd.Execute()
	})

	// Endpoint check for the live job should appear (will fail since localhost:3000 isn't running).
	if !strings.Contains(out, "endpoint:my-job") {
		t.Errorf("expected endpoint check for my-job in output, got: %s", out)
	}
}

func TestDoctorCheckEndpoints_StandaloneNoJobsReturnsWarn(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
		"GET /v1/system/capabilities": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"code_deploy_enabled": true})
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = "json"
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{"--check-endpoints"})

	out := captureStateOutput(t, state, func() {
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "endpoints_live") {
		t.Errorf("expected endpoints_live check in output, got: %s", out)
	}
}

func TestDoctorCommand_HasCheckEndpointsFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newDoctorCommand(state)
	if cmd.Flags().Lookup("check-endpoints") == nil {
		t.Error("expected --check-endpoints flag on doctor command")
	}
}
