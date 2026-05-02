package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestVerify_AllChecksPass(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /health/ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	cmd := newVerifyCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "health") {
		t.Fatalf("expected health check in output, got: %s", out)
	}
}

func TestVerify_HealthFails(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusServiceUnavailable, "unhealthy")
		},
		"GET /health/ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
	})

	state := newTestState(t, srv)
	cmd := newVerifyCommand(state)

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "verify failed") {
			t.Fatalf("expected verify failed error, got: %v", err)
		}
	})
}
