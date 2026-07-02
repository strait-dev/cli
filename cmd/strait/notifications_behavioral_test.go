package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// testChannelFixture returns a fresh NotificationChannel per call so parallel
// tests do not share the same backing array for Config (json.RawMessage is a
// []byte slice).
func testChannelFixture() types.NotificationChannel {
	return types.NotificationChannel{
		ID:        "ch-1",
		ProjectID: "proj-test",
		Name:      "oncall",
		Type:      "slack",
		Config:    json.RawMessage(`{"webhook_url":"https://hooks.slack.com/x"}`),
		Enabled:   true,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestNotificationsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/notification-channels": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondJSON(t, w, http.StatusOK, []types.NotificationChannel{testChannelFixture()})
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "oncall") {
		t.Fatalf("expected channel name in output: %s", out)
	}
	// list intentionally omits the Config field rather than including a
	// masked placeholder — assert the secret is absent regardless.
	if strings.Contains(out, "hooks.slack.com") {
		t.Fatalf("expected list output to omit config secrets, got: %s", out)
	}
}

func TestNotificationsGet_MasksConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/notification-channels/ch-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testChannelFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsGetCommand(state)
	cmd.SetArgs([]string{"ch-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "hooks.slack.com") {
		t.Fatalf("expected config to be masked, got: %s", out)
	}
	if !strings.Contains(out, "********") {
		t.Fatalf("expected mask placeholder, got: %s", out)
	}
}

func TestNotificationsGet_RevealUnmasks(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/notification-channels/ch-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testChannelFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsGetCommand(state)
	cmd.SetArgs([]string{"ch-1", "--reveal"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "hooks.slack.com") {
		t.Fatalf("expected revealed config in output: %s", out)
	}
}

func TestNotificationsCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newNotificationsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--type", "slack", "--config-json", `{"webhook_url":"x"}`})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestNotificationsCreate_RequiresType(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newNotificationsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "oncall", "--config-json", `{"x":1}`})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--type is required") {
		t.Fatalf("expected --type error, got: %v", err)
	}
}

func TestNotificationsCreate_RejectsInvalidConfigJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newNotificationsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "oncall", "--type", "slack", "--config-json", "not-json"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected JSON validation error, got: %v", err)
	}
}

func TestNotificationsCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/notification-channels": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got map[string]any
			readJSONBody(t, r, &got)
			if got["name"] != "oncall" {
				t.Errorf("name: got %q, want %q", got["name"], "oncall")
			}
			if got["channel_type"] != "slack" {
				t.Errorf("channel_type: got %q, want %q", got["channel_type"], "slack")
			}
			if _, ok := got["type"]; ok {
				t.Errorf("request included legacy type field: %#v", got)
			}
			respondJSON(t, w, http.StatusCreated, testChannelFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "oncall",
		"--type", "slack",
		"--config-json", `{"webhook_url":"https://hooks.slack.com/x"}`,
	})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotificationsUpdate_RequiresAtLeastOneFlag(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newNotificationsUpdateCommand(state)
	cmd.SetArgs([]string{"ch-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at-least-one-flag error, got: %v", err)
	}
}

func TestNotificationsDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/notification-channels/ch-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsDeleteCommand(state)
	cmd.SetArgs([]string{"ch-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
