package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// testDrainFixture returns a fresh LogDrain per call so parallel tests do not
// share the same backing array for Config (json.RawMessage is a []byte slice).
func testDrainFixture() types.LogDrain {
	return types.LogDrain{
		ID:        "drain-1",
		ProjectID: "proj-test",
		Name:      "prod-dd",
		Type:      "datadog",
		Config:    json.RawMessage(`{"api_key":"secret"}`),
		Enabled:   true,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestLogDrainsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.LogDrain{testDrainFixture()})
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
	// list intentionally omits Config rather than masking — assert the
	// secret is absent regardless.
	if strings.Contains(out, "secret") {
		t.Fatalf("expected list output to omit config secrets, got: %s", out)
	}
}

func TestLogDrainsGet_MasksConfig(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains/drain-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testDrainFixture())
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
	if !strings.Contains(out, "********") {
		t.Fatalf("expected mask placeholder, got: %s", out)
	}
}

func TestLogDrainsGet_RevealUnmasks(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains/drain-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testDrainFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsGetCommand(state)
	cmd.SetArgs([]string{"drain-1", "--reveal"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "secret") {
		t.Fatalf("expected revealed config in output: %s", out)
	}
}

func TestLogDrainsCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--type", "datadog", "--endpoint-url", "https://e", "--auth-type", "none"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestLogDrainsCreate_RejectsInvalidAuthConfigJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "x", "--type", "datadog", "--endpoint-url", "https://e", "--auth-type", "api_key", "--auth-config", "nope"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected JSON validation error, got: %v", err)
	}
}

func TestLogDrainsCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/log-drains": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got struct {
				Name        string `json:"name"`
				Type        string `json:"drain_type"`
				EndpointURL string `json:"endpoint_url"`
				AuthType    string `json:"auth_type"`
			}
			readJSONBody(t, r, &got)
			if got.Name != "prod-dd" {
				t.Errorf("name: got %q, want %q", got.Name, "prod-dd")
			}
			if got.Type != "datadog" {
				t.Errorf("drain_type: got %q, want %q", got.Type, "datadog")
			}
			if got.EndpointURL == "" || got.AuthType == "" {
				t.Errorf("endpoint_url/auth_type missing: %+v", got)
			}
			respondJSON(t, w, http.StatusCreated, testDrainFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newLogDrainsCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--name", "prod-dd",
		"--type", "datadog",
		"--endpoint-url", "https://http-intake.logs.datadoghq.com",
		"--auth-type", "api_key",
		"--auth-config", `{"api_key":"x"}`,
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
