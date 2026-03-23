package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
)

var testSecret = client.ServerSecret{
	ID:          "sec-1",
	ProjectID:   "proj-test",
	SecretKey:   "DB_PASSWORD",
	Environment: "production",
	CreatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

func TestSecretsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/secrets": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []client.ServerSecret{testSecret})
		},
	})

	state := newTestState(t, srv)
	cmd := newSecretsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "DB_PASSWORD") {
		t.Fatalf("expected DB_PASSWORD in output, got: %s", out)
	}
}

func TestSecretsList_FilterEnv(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/secrets": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "environment", "production")
			respondPaginated(t, w, http.StatusOK, []client.ServerSecret{testSecret})
		},
	})

	state := newTestState(t, srv)
	cmd := newSecretsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--environment", "production"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "DB_PASSWORD") {
		t.Fatalf("expected DB_PASSWORD in output, got: %s", out)
	}
}

func TestSecretsCreate_Success(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/secrets": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, testSecret)
		},
	})

	state := newTestState(t, srv)
	cmd := newSecretsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--key", "DB_PASSWORD",
		"--value", "s3cret",
	})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if receivedBody["secret_key"] != "DB_PASSWORD" {
		t.Fatalf("expected secret_key=DB_PASSWORD in body, got: %v", receivedBody)
	}
	if receivedBody["secret_value"] != "s3cret" {
		t.Fatalf("expected secret_value=s3cret in body, got: %v", receivedBody)
	}
	if !strings.Contains(out, "DB_PASSWORD") {
		t.Fatalf("expected DB_PASSWORD in output, got: %s", out)
	}
}

func TestSecretsCreate_MissingKey(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newSecretsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--value", "s3cret"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--key") {
		t.Fatalf("expected --key required error, got: %v", err)
	}
}

func TestSecretsDelete_WithYes(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/secrets/sec-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
	})

	state := newTestState(t, srv)
	cmd := newSecretsDeleteCommand(state)
	cmd.SetArgs([]string{"sec-1", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestSecretsDelete_CIBlocksPrompt(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	state.opts.ciMode = true
	cmd := newSecretsDeleteCommand(state)
	cmd.SetArgs([]string{"sec-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "CI mode") {
		t.Fatalf("expected CI mode error, got: %v", err)
	}
}
