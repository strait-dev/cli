package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestDiagnose_RunsChecks(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
	})

	state := newTestState(t, srv)
	cmd := newDiagnoseCommand(state)
	cmd.SetArgs([]string{})

	// Diagnose may report failing checks due to env vars (DATABASE_URL etc.)
	// not being set in CI. We just verify it runs and produces output.
	out := captureCommandOutput(t, func() {
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "server configured") {
		t.Fatalf("expected 'server configured' check in output, got: %s", out)
	}
	if !strings.Contains(out, "health") {
		t.Fatalf("expected 'health' check in output, got: %s", out)
	}
}

func TestDiagnose_ServerUnreachable(t *testing.T) {
	t.Parallel()

	// Use a server that returns errors for health and stats
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusServiceUnavailable, "down")
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusUnauthorized, "unauthorized")
		},
	})

	state := newTestState(t, srv)
	cmd := newDiagnoseCommand(state)
	cmd.SetArgs([]string{})

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failing checks") {
			t.Fatalf("expected failing checks error, got: %v", err)
		}
	})
}

func TestDiagnose_MissingConfig(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.serverURL = ""
	state.opts.apiKey = ""
	state.opts.projectID = ""
	cmd := newDiagnoseCommand(state)
	cmd.SetArgs([]string{})

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failing checks") {
			t.Fatalf("expected failing checks, got: %v", err)
		}
	})
}

func TestDiagnoseCheck_Helper(t *testing.T) {
	t.Parallel()

	check := diagnoseCheck("test-check", true, "all good", "no fix needed")
	if check["check"] != "test-check" {
		t.Fatalf("expected check name, got: %v", check)
	}
	if check["ok"] != true {
		t.Fatal("expected ok=true")
	}
	if check["detail"] != "all good" {
		t.Fatalf("expected detail, got: %v", check["detail"])
	}
}

func TestBoolString(t *testing.T) {
	t.Parallel()

	if boolString(true) != "set" {
		t.Fatal("expected 'set' for true")
	}
	if boolString(false) != "missing" {
		t.Fatal("expected 'missing' for false")
	}
}
