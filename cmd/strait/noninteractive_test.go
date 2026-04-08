package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestNonInteractiveFlag_RegisteredOnRoot(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	flag := cmd.PersistentFlags().Lookup("non-interactive")
	if flag == nil {
		t.Fatal("--non-interactive flag not registered on root command")
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected default false, got %q", flag.DefValue)
	}
}

func TestRequireConfirmation_BlocksInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.nonInteractive = true

	err := requireConfirmation(state, "are you sure?", false)
	if err == nil {
		t.Fatal("expected error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("expected non-interactive message, got: %v", err)
	}
}

func TestRequireConfirmation_PassesWithYesFlag(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.nonInteractive = true

	// --yes bypasses non-interactive check
	if err := requireConfirmation(state, "are you sure?", true); err != nil {
		t.Fatalf("expected nil when yes=true, got: %v", err)
	}
}

func TestRequireConfirmation_CIModeAlsoBlocks(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	// ciMode is already true in newTestState; nonInteractive should be set too
	state.opts.ciMode = true
	state.opts.nonInteractive = true

	err := requireConfirmation(state, "continue?", false)
	if err == nil {
		t.Fatal("expected error when ciMode and nonInteractive are both true")
	}
}

func TestNonInteractiveFlagShortCircuitsJobDelete(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-abc": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id":   "job-abc",
				"name": "Test Job",
				"slug": "test-job",
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.nonInteractive = true

	// delete without --yes in non-interactive mode must fail fast
	cmd := newJobsCommand(state)
	cmd.SetArgs([]string{"delete", "job-abc"})

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "non-interactive") {
			t.Fatalf("expected non-interactive error, got: %v", err)
		}
	})
}

func TestNonInteractiveFlagWithYesAllowsJobDelete(t *testing.T) {
	t.Parallel()

	deleted := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-abc": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id":   "job-abc",
				"name": "Test Job",
				"slug": "test-job",
			})
		},
		"DELETE /v1/jobs/job-abc": func(w http.ResponseWriter, _ *http.Request) {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		},
	})

	state := newTestState(t, srv)
	state.opts.nonInteractive = true

	cmd := newJobsCommand(state)
	cmd.SetArgs([]string{"delete", "job-abc", "--yes"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error with --yes: %v", err)
		}
	})

	if !deleted {
		t.Fatal("expected DELETE request to have been made")
	}
}
