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
	out := captureStateOutput(t, state, func() {
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
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected project required error, got: %v", err)
	}
}

// TestTriggersList_UsesResolvedProject verifies that `triggers list` resolves
// the project from the shared resolver (config/env/context) and no longer
// requires an explicit --project flag.
func TestTriggersList_UsesResolvedProject(t *testing.T) {
	t.Parallel()
	var hit bool
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/events": func(w http.ResponseWriter, r *http.Request) {
			hit = true
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.EventTrigger{})
		},
	})
	state := newTestState(t, srv) // newTestState sets project to proj-test
	cmd := newTriggersListCommand(state)
	cmd.SetArgs([]string{})
	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatal("expected the events endpoint to be called with the resolved project")
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
	out := captureStateOutput(t, state, func() {
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
	out := captureStateOutput(t, state, func() {
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
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "5") {
		t.Fatalf("expected deleted count in output, got: %s", out)
	}
}

func TestTriggersSend_Raw(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/events/dispatch": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &receivedBody)
			w.WriteHeader(http.StatusAccepted)
		},
	})
	state := newTestState(t, srv)
	cmd := newTriggersSendCommand(state)
	cmd.SetArgs([]string{"orders.created", "--raw", "--project", "proj-test", "--payload", `{"order_id":"abc"}`})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if receivedBody["source"] != "orders.created" {
		t.Fatalf("expected source=orders.created, got: %v", receivedBody)
	}
	if receivedBody["project_id"] != "proj-test" {
		t.Fatalf("expected project_id=proj-test, got: %v", receivedBody)
	}
	payload, _ := receivedBody["payload"].(map[string]any)
	if payload == nil || payload["order_id"] != "abc" {
		t.Fatalf("expected payload.order_id=abc, got: %v", receivedBody["payload"])
	}
	if !strings.Contains(out, "orders.created") {
		t.Fatalf("expected event key in output: %s", out)
	}
}

func TestTriggersSend_RawRequiresProject(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newTriggersSendCommand(state)
	cmd.SetArgs([]string{"orders.created", "--raw"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--project is required") {
		t.Fatalf("expected project-required error, got: %v", err)
	}
}

func TestTriggersStream_EmitsNewTriggers(t *testing.T) {
	t.Parallel()

	second := testTrigger
	second.ID = "trig-2"
	second.EventKey = "next-event"

	calls := 0
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/events": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			calls++
			switch calls {
			case 1:
				respondPaginated(t, w, http.StatusOK, []types.EventTrigger{testTrigger})
			default:
				respondPaginated(t, w, http.StatusOK, []types.EventTrigger{testTrigger, second})
			}
		},
	})

	// Replace the blocking sleep with a cancel on the second poll so we exit
	// the stream loop after observing the new trigger.
	state := newTestState(t, srv)
	cmd := newTriggersStreamCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--interval", "1ms"})

	origSleep := triggersStreamSleep
	t.Cleanup(func() { triggersStreamSleep = origSleep })

	sleepCalls := 0
	triggersStreamSleep = func(_ time.Duration) {
		sleepCalls++
		if sleepCalls >= 2 {
			panic("stream-done") // unwind out of the infinite loop
		}
	}

	out := captureStateOutput(t, state, func() {
		defer func() {
			if r := recover(); r != nil && r != "stream-done" {
				panic(r)
			}
		}()
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "trig-2") || !strings.Contains(out, "next-event") {
		t.Fatalf("expected new trigger in output: %s", out)
	}
	if strings.Contains(out, "trig-1") {
		t.Fatalf("did not expect seeded trigger in output: %s", out)
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
