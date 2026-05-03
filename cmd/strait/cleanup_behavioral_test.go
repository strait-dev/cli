package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCleanup_MissingOlderThan(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newCleanupCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "runs-older-than") {
		t.Fatalf("expected runs-older-than error, got: %v", err)
	}
}

func TestCleanup_DryRun(t *testing.T) {
	t.Parallel()

	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "run-old", "status": "completed", "created_at": oldTime.Format(time.RFC3339)},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newCleanupCommand(state)
	cmd.SetArgs([]string{"--runs-older-than", "720h", "--dry-run"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "dry_run") {
		t.Fatalf("expected dry_run in output, got: %s", out)
	}
}

func TestCleanup_NoMatches(t *testing.T) {
	t.Parallel()

	// Return runs that are too recent to match
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "run-recent", "status": "completed", "created_at": time.Now().Format(time.RFC3339)},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newCleanupCommand(state)
	cmd.SetArgs([]string{"--runs-older-than", "720h"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "no runs matched") {
		t.Fatalf("expected no-match message, got: %s", out)
	}
}

func TestCleanup_CIBlocksPrompt(t *testing.T) {
	t.Parallel()

	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "run-old", "status": "completed", "created_at": oldTime.Format(time.RFC3339)},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.ciMode = true
	cmd := newCleanupCommand(state)
	cmd.SetArgs([]string{"--runs-older-than", "720h"})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "non-interactive") {
			t.Fatalf("expected non-interactive error, got: %v", err)
		}
	})
}
