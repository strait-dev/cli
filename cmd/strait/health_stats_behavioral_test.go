package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHealth_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
	})
	state := newTestState(t, srv)
	cmd := newHealthCommand(state)
	cmd.SetArgs([]string{})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected ok in output, got: %s", out)
	}
}

func TestHealthReady_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health/ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
	})
	state := newTestState(t, srv)
	cmd := newHealthCommand(state)
	cmd.SetArgs([]string{"--ready"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected ok in output, got: %s", out)
	}
}

func TestStats_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{
				"queued": 5, "executing": 2, "delayed": 1,
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newStatsCommand(state)
	cmd.SetArgs([]string{})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "queued") {
		t.Fatalf("expected stats in output, got: %s", out)
	}
}
