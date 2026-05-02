package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testDrain = types.LogDrain{
	ID:        "drain-1",
	ProjectID: "proj-test",
	Name:      "prod-dd",
	Type:      "datadog",
	Config:    json.RawMessage(`{"api_key":"secret"}`),
	Enabled:   true,
	CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
}

func TestLogDrainsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.LogDrain{testDrain})
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "prod-dd") {
		t.Fatalf("expected drain name in output: %s", out)
	}
}

func TestLogDrainsGet_MasksConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains/drain-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testDrain)
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsGetCommand(state)
	cmd.SetArgs([]string{"drain-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "secret") {
		t.Fatalf("expected api_key to be masked, got: %s", out)
	}
}

func TestLogDrainsCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--type", "datadog", "--config-json", `{"api_key":"x"}`})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestLogDrainsCreate_RejectsInvalidConfigJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "x", "--type", "datadog", "--config-json", "nope"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected JSON validation error, got: %v", err)
	}
}

func TestLogDrainsCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/log-drains": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusCreated, testDrain)
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "prod-dd",
		"--type", "datadog",
		"--config-json", `{"api_key":"x","site":"us"}`,
	})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogDrainsUpdate_RequiresAtLeastOneFlag(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newLogDrainsUpdateCommand(state)
	cmd.SetArgs([]string{"drain-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at-least-one-flag error, got: %v", err)
	}
}

func TestLogDrainsDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/log-drains/drain-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsDeleteCommand(state)
	cmd.SetArgs([]string{"drain-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
