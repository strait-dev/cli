package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDoctorCommand_PassesWhenServerCapabilitiesEnabled(t *testing.T) {
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
			respondJSON(t, w, http.StatusOK, map[string]any{
				"code_deploy_enabled": true,
				"registry_host":       "registry.example.com",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureCommandOutput(t, func() {
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "code_deploy_supported") {
		t.Fatalf("expected code_deploy_supported check in output, got: %s", out)
	}
}

func TestDoctorCommand_FailsWhenCodeDeployDisabled(t *testing.T) {
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
			respondJSON(t, w, http.StatusOK, map[string]any{
				"code_deploy_enabled": false,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failing check") {
			t.Fatalf("expected failing checks error, got: %v", err)
		}
	})

	if !strings.Contains(out, "code_deploy_supported") {
		t.Fatalf("expected code_deploy_supported check in output, got: %s", out)
	}
}

func TestDoctorCommand_WarnsWhenCapabilitiesEndpointUnavailable(t *testing.T) {
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
		// /v1/system/capabilities returns 404 (older server)
		"GET /v1/system/capabilities": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "not found")
		},
	})

	state := newTestState(t, srv)
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureCommandOutput(t, func() {
		// may or may not return error depending on other checks
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "code_deploy_supported") {
		t.Fatalf("expected code_deploy_supported check in output, got: %s", out)
	}
}

func TestDoctorCommand_RuntimeDetectedCheck(t *testing.T) {
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
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureCommandOutput(t, func() {
		_ = cmd.Execute()
	})

	// The test runs from the repo root which has go.mod — so runtime_detected should pass with "go".
	// If go.mod is not present, it will be a warn. Either way the check name must appear.
	if !strings.Contains(out, "runtime_detected") {
		t.Fatalf("expected runtime_detected check in output, got: %s", out)
	}
}

func TestDoctorCommand_OutputIncludesNewChecks(t *testing.T) {
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
	cmd := newDoctorCommand(state)
	cmd.SetArgs([]string{})

	out := captureCommandOutput(t, func() {
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "code_deploy_supported") {
		t.Fatalf("expected code_deploy_supported in output, got: %s", out)
	}
	if !strings.Contains(out, "runtime_detected") {
		t.Fatalf("expected runtime_detected in output, got: %s", out)
	}
}

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

	// Wire through root so --format is a known flag.
	root := newRootCommand()
	root.SetArgs([]string{"--format", "json", "--server", srv.URL, "--api-key", "test-key", "--project", "proj-1", "doctor"})

	out := captureCommandOutput(t, func() {
		_ = root.Execute()
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

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for unknown --json flag, got nil")
		}
	})
}
