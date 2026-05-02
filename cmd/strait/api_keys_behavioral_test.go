package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testAPIKey = types.APIKey{
	ID:        "key-1",
	ProjectID: "proj-test",
	Name:      "CI Key",
	KeyPrefix: "sk_test_",
	Scopes:    []string{"jobs:read", "jobs:write"},
	CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
}

func TestAPIKeysList_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/api-keys": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.APIKey{testAPIKey})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPIKeysListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "key-1") {
		t.Fatalf("expected key-1 in output, got: %s", out)
	}
}

func TestAPIKeysCreate_Success(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/api-keys": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, map[string]any{
				"id": "key-new", "project_id": "proj-test", "name": "New Key",
				"key": "sk_test_abc123", "key_prefix": "sk_test_",
				"scopes": []string{"jobs:read"}, "created_at": "2026-01-15T00:00:00Z",
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPIKeysCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "New Key", "--scopes", "jobs:read"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if receivedBody["name"] != "New Key" {
		t.Fatalf("expected name=New Key in body, got: %v", receivedBody)
	}
	if !strings.Contains(out, "key-new") {
		t.Fatalf("expected key-new in output, got: %s", out)
	}
}

func TestAPIKeysCreate_MissingFields(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newAPIKeysCreateCommand(state)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestAPIKeysRevoke_Success(t *testing.T) {
	t.Parallel()
	revokeCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/api-keys/key-1": func(w http.ResponseWriter, _ *http.Request) {
			revokeCalled = true
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPIKeysRevokeCommand(state)
	cmd.SetArgs([]string{"key-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revokeCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestAPIKeysRotate_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/api-keys/key-1/rotate": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"old_key_id": "key-1", "new_key_id": "key-2",
				"project_id": "proj-test", "name": "CI Key",
				"key": "sk_test_new", "key_prefix": "sk_test_",
				"scopes": []string{"jobs:read"}, "created_at": "2026-01-15T00:00:00Z",
				"grace_expires_at": "2026-01-15T01:00:00Z",
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPIKeysRotateCommand(state)
	cmd.SetArgs([]string{"key-1", "--grace-period-minutes", "60"})
	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "key-1") {
		t.Fatalf("expected key-1 in output, got: %s", out)
	}
}
