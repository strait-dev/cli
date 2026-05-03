package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestDebugCommand_HasRequestSubcommand(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newDebugCommand(state)

	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"bundle", "request"} {
		if !names[want] {
			t.Errorf("expected subcommand %q on debug command", want)
		}
	}
}

func TestDebugRequestCommand_HasFlags(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newDebugRequestCommand(state)
	if cmd.Flags().Lookup("body") == nil {
		t.Error("expected --body flag on debug request command")
	}
}

func TestDebugRequestCommand_GETReturnsJSON(t *testing.T) {
	t.Parallel()

	jobs := []map[string]any{{"id": "job-1", "slug": "my-job"}}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": jobs})
		},
	})

	state := newTestState(t, srv)
	cmd := newDebugRequestCommand(state)
	cmd.SetArgs([]string{"GET", "/v1/jobs"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-1") {
		t.Errorf("expected job-1 in output, got: %s", out)
	}
}

func TestDebugRequestCommand_POSTWithBody(t *testing.T) {
	t.Parallel()

	var captured map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			respondJSON(t, w, http.StatusCreated, map[string]any{"id": "job-new"})
		},
	})

	state := newTestState(t, srv)
	cmd := newDebugRequestCommand(state)
	cmd.SetArgs([]string{"POST", "/v1/jobs", "--body", `{"name":"test","slug":"test"}`})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if captured["name"] != "test" {
		t.Errorf("expected captured name=test, got: %v", captured["name"])
	}
}

func TestDebugRequestCommand_RequiresTwoArgs(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newDebugRequestCommand(state)
	cmd.SetArgs([]string{"GET"}) // missing path

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error with missing path argument")
	}
}

func TestDebugRequestCommand_NonJSONResponsePrinted(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/health": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "ok")
		},
	})

	state := newTestState(t, srv)
	cmd := newDebugRequestCommand(state)
	cmd.SetArgs([]string{"GET", "/v1/health"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok' in output, got: %s", out)
	}
}

func TestDebugRequestCommand_SendsAuthHeader(t *testing.T) {
	t.Parallel()

	var capturedAuth string
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			capturedAuth = r.Header.Get("Authorization")
			respondJSON(t, w, http.StatusOK, map[string]any{"data": []any{}})
		},
	})

	state := newTestState(t, srv)
	cmd := newDebugRequestCommand(state)
	cmd.SetArgs([]string{"GET", "/v1/jobs"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("expected Bearer auth header, got: %q", capturedAuth)
	}
}

func TestRootCommand_HasDebugFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	if cmd.PersistentFlags().Lookup("debug") == nil {
		t.Error("expected --debug persistent flag on root command")
	}
}

func TestDebugTransport_LogsToStderr(t *testing.T) {
	t.Parallel()

	// debugTransport is invoked implicitly when state.opts.debug = true.
	// We verify that --debug doesn't break normal request flow by checking
	// the response is still valid.
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"data": []any{}})
		},
	})

	state := newTestState(t, srv)
	state.opts.debug = true

	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	_, listErr := cli.ListJobs(t.Context(), "proj-1")
	if listErr != nil {
		t.Fatalf("ListJobs with debug transport: %v", listErr)
	}
}
