package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func testNotificationFixture() types.NotificationChannel {
	return types.NotificationChannel{
		ID:        "channel-1",
		ProjectID: "proj-test",
		Name:      "ops",
		Type:      "slack",
		Config:    json.RawMessage(`{"webhook_url":"secret"}`),
		Enabled:   true,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestNotificationsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/notification-channels": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.NotificationChannel{testNotificationFixture()})
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

	if !strings.Contains(out, "ops") {
		t.Fatalf("expected notification name in output: %s", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("expected list output to omit config: %s", out)
	}
}

func TestNotificationsGet_MasksConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/notification-channels/channel-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testNotificationFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newNotificationsGetCommand(state)
	cmd.SetArgs([]string{"channel-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "secret") || !strings.Contains(out, "********") {
		t.Fatalf("expected masked config, got: %s", out)
	}
}

func TestNotificationsCreateAndUpdate(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/notification-channels": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				ProjectID string          `json:"project_id"`
				Name      string          `json:"name"`
				Type      string          `json:"channel_type"`
				Config    json.RawMessage `json:"config"`
			}
			readJSONBody(t, r, &got)
			if got.Name != "ops" || got.Type != "slack" || !json.Valid(got.Config) {
				t.Errorf("unexpected create payload: %+v", got)
			}
			respondJSON(t, w, http.StatusCreated, testNotificationFixture())
		},
		"PATCH /v1/notification-channels/channel-1": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				Name *string `json:"name"`
			}
			readJSONBody(t, r, &got)
			if got.Name == nil || *got.Name != "pager" {
				t.Errorf("unexpected update payload: %+v", got)
			}
			channel := testNotificationFixture()
			channel.Name = "pager"
			respondJSON(t, w, http.StatusOK, channel)
		},
		"DELETE /v1/notification-channels/channel-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	createCmd := newNotificationsCreateCommand(state)
	createCmd.SetArgs([]string{"--project", "proj-test", "--name", "ops", "--type", "slack", "--config-json", `{"webhook_url":"x"}`})
	if err := captureAndExec(t, state, createCmd); err != nil {
		t.Fatalf("create: %v", err)
	}

	updateCmd := newNotificationsUpdateCommand(state)
	updateCmd.SetArgs([]string{"channel-1", "--name", "pager"})
	if err := captureAndExec(t, state, updateCmd); err != nil {
		t.Fatalf("update: %v", err)
	}

	deleteCmd := newNotificationsDeleteCommand(state)
	deleteCmd.SetArgs([]string{"channel-1", "--yes"})
	if err := captureAndExec(t, state, deleteCmd); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
