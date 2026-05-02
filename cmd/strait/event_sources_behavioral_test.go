package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testEventSource = types.EventSource{
	ID:        "src-1",
	ProjectID: "proj-test",
	Name:      "Kafka Source",
	Slug:      "kafka-source",
	Type:      "kafka",
	Enabled:   true,
	CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
}

func TestEventSourcesList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/event-sources": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.EventSource{testEventSource})
		},
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "kafka-source") {
		t.Fatalf("expected slug in output: %s", out)
	}
}

func TestEventSourcesGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/event-sources/src-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testEventSource)
		},
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesGetCommand(state)
	cmd.SetArgs([]string{"src-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventSourcesCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newEventSourcesCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--slug", "x", "--type", "kafka"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestEventSourcesCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/event-sources": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusCreated, testEventSource)
		},
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "Kafka Source",
		"--slug", "kafka-source",
		"--type", "kafka",
		"--config-json", `{"broker":"localhost"}`,
	})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventSourcesCreate_RejectsInvalidConfigJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newEventSourcesCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "x",
		"--slug", "x",
		"--type", "kafka",
		"--config-json", `not-json`,
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected JSON validation error, got: %v", err)
	}
}

func TestEventSourcesUpdate_RequiresAtLeastOneFlag(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/event-sources/src-1":   func(w http.ResponseWriter, _ *http.Request) { respondError(t, w, http.StatusNotFound, "nope") },
		"PATCH /v1/event-sources/src-1": func(_ http.ResponseWriter, _ *http.Request) { t.Fatal("PATCH should not be reached") },
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesUpdateCommand(state)
	cmd.SetArgs([]string{"src-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at-least-one-flag error, got: %v", err)
	}
}

func TestEventSourcesDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/event-sources/src-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testEventSource)
		},
		"DELETE /v1/event-sources/src-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesDeleteCommand(state)
	cmd.SetArgs([]string{"src-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
