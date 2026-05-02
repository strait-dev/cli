package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testWebhook = types.Webhook{
	ID:        "webhook-1",
	ProjectID: "proj-test",
	URL:       "https://example.com/hook",
	Events:    []string{"job.run.completed"},
	Active:    true,
	CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
}

func TestWebhooksList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/webhooks": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.Webhook{testWebhook})
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "webhook-1") {
		t.Fatalf("expected webhook-1 in output: %s", out)
	}
}

func TestWebhooksList_NoProject(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newWebhooksListCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected project ID error, got: %v", err)
	}
}

func TestWebhooksGet_ByID(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/webhooks/webhook-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWebhook)
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksGetCommand(state)
	cmd.SetArgs([]string{"webhook-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "webhook-1") {
		t.Fatalf("expected webhook-1 in output: %s", out)
	}
}

func TestWebhooksCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/webhooks": func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, "POST")
			respondJSON(t, w, http.StatusCreated, testWebhook)
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--url", "https://example.com/hook", "--event", "job.run.completed"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksCreate_RequiresURL(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWebhooksCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--event", "x"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--url is required") {
		t.Fatalf("expected --url required error, got: %v", err)
	}
}

func TestWebhooksCreate_RequiresEvent(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWebhooksCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--url", "https://example.com/hook"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--event is required") {
		t.Fatalf("expected --event required error, got: %v", err)
	}
}

func TestWebhooksUpdate_RequiresAtLeastOneFlag(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWebhooksUpdateCommand(state)
	cmd.SetArgs([]string{"webhook-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one update flag") {
		t.Fatalf("expected update flag error, got: %v", err)
	}
}

func TestWebhooksUpdate_PatchURL(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PATCH /v1/webhooks/webhook-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testWebhook)
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksUpdateCommand(state)
	cmd.SetArgs([]string{"webhook-1", "--url", "https://new.example.com/hook"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksDelete_RequiresYes(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newWebhooksDeleteCommand(state)
	cmd.SetArgs([]string{"webhook-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestWebhooksDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/webhooks/webhook-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksDeleteCommand(state)
	cmd.SetArgs([]string{"webhook-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksDeliveries_Success(t *testing.T) {
	t.Parallel()

	delivery := types.WebhookDelivery{
		ID:           "delivery-1",
		WebhookID:    "webhook-1",
		EventType:    "job.run.completed",
		Status:       "succeeded",
		StatusCode:   200,
		AttemptCount: 1,
		RequestedAt:  time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/webhooks/webhook-1/deliveries": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.WebhookDelivery{delivery})
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksDeliveriesCommand(state)
	cmd.SetArgs([]string{"webhook-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "delivery-1") {
		t.Fatalf("expected delivery-1 in output: %s", out)
	}
}

func TestWebhooksRetry_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/webhook-deliveries/delivery-1/retry": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.WebhookDelivery{
				ID: "delivery-1", WebhookID: "webhook-1", Status: "succeeded",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksRetryCommand(state)
	cmd.SetArgs([]string{"delivery-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksTest_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/webhooks/webhook-1/test": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"ok": true})
		},
	})

	state := newTestState(t, srv)
	cmd := newWebhooksTestCommand(state)
	cmd.SetArgs([]string{"webhook-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// captureAndExec executes a command, captures any output to the state buffer,
// and returns the command's error. Used by tests that don't need to inspect
// the output but do want to ensure the buffer is reset between invocations.
func captureAndExec(t *testing.T, state *appState, cmd interface {
	Execute() error
}) error {
	t.Helper()
	var execErr error
	captureStateOutput(t, state, func() {
		execErr = cmd.Execute()
	})
	return execErr
}
