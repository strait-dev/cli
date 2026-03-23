package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testTrigger = types.EventTrigger{
	ID:          "trig-1",
	EventKey:    "my-event-key",
	ProjectID:   "proj-test",
	SourceType:  "workflow_step",
	Status:      "waiting",
	TimeoutSecs: 3600,
	RequestedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	ExpiresAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
}

func TestTriggersList_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/events": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.EventTrigger{testTrigger})
		},
	})
	state := newTestState(t, srv)
	cmd := newTriggersListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "trig-1") {
		t.Fatalf("expected trig-1 in output, got: %s", out)
	}
}

func TestTriggersList_RequiresProject(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newTriggersListCommand(state)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--project is required") {
		t.Fatalf("expected project required error, got: %v", err)
	}
}

func TestTriggersGet_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/events/my-event-key": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testTrigger)
		},
	})
	state := newTestState(t, srv)
	cmd := newTriggersGetCommand(state)
	cmd.SetArgs([]string{"my-event-key"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "my-event-key") {
		t.Fatalf("expected event key in output, got: %s", out)
	}
}

func TestTriggersSend_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/events/my-event-key/send": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testTrigger)
		},
	})
	state := newTestState(t, srv)
	cmd := newTriggersSendCommand(state)
	cmd.SetArgs([]string{"my-event-key", "--payload", `{"data":"value"}`})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "my-event-key") {
		t.Fatalf("expected event key in output, got: %s", out)
	}
}

func TestTriggersPurge_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/events/purge": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"deleted": float64(5)})
		},
	})
	state := newTestState(t, srv)
	cmd := newTriggersPurgeCommand(state)
	cmd.SetArgs([]string{"--older-than", "30"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "5") {
		t.Fatalf("expected deleted count in output, got: %s", out)
	}
}

func TestTriggersPurge_InvalidOlderThan(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newTriggersPurgeCommand(state)
	cmd.SetArgs([]string{"--older-than", "0"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), ">= 1 day") {
		t.Fatalf("expected older-than error, got: %v", err)
	}
}
