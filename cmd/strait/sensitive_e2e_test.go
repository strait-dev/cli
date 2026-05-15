package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/types"
)

// TestEnvironmentsGet_MasksVariablesByDefault asserts that getting an
// environment in JSON mode hides variable values unless --reveal is set.
func TestEnvironmentsGet_MasksVariablesByDefault(t *testing.T) {
	t.Parallel()

	env := types.Environment{
		ID:        "env-1",
		ProjectID: "proj-test",
		Name:      "Production",
		Slug:      "prod",
		Variables: map[string]string{
			"DATABASE_URL": "postgres://user:hunter2@db.example.com/prod",
			"API_KEY":      "sk_live_xyz",
		},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/environments/env-1": func(w http.ResponseWriter, r *http.Request) {
			respondJSON(t, w, http.StatusOK, env)
		},
	})
	state := newTestState(t, srv)
	cmd := newEnvironmentsGetCommand(state)
	cmd.SetArgs([]string{"env-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	if strings.Contains(out, "hunter2") {
		t.Fatalf("output leaked DATABASE_URL secret:\n%s", out)
	}
	if strings.Contains(out, "sk_live_xyz") {
		t.Fatalf("output leaked API_KEY secret:\n%s", out)
	}
	if !strings.Contains(out, sensitiveMask) {
		t.Fatalf("output missing mask placeholder:\n%s", out)
	}
}

func TestEnvironmentsGet_RevealsVariablesWithFlag(t *testing.T) {
	t.Parallel()

	env := types.Environment{
		ID:   "env-1",
		Slug: "prod",
		Variables: map[string]string{
			"API_KEY": "sk_live_xyz",
		},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/environments/env-1": func(w http.ResponseWriter, r *http.Request) {
			respondJSON(t, w, http.StatusOK, env)
		},
	})
	state := newTestState(t, srv)
	cmd := newEnvironmentsGetCommand(state)
	cmd.SetArgs([]string{"env-1", "--reveal"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	if !strings.Contains(out, "sk_live_xyz") {
		t.Fatalf("--reveal output missing API_KEY:\n%s", out)
	}
}

func TestEnvironmentsVariables_MasksByDefault(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/environments/env-1": func(w http.ResponseWriter, r *http.Request) {
			respondJSON(t, w, http.StatusOK, types.Environment{ID: "env-1", Slug: "prod"})
		},
		"GET /v1/environments/env-1/variables": func(w http.ResponseWriter, r *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{
				"PG_PASSWORD": "very-secret-password",
				"OTHER":       "value",
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newEnvironmentsVariablesCommand(state)
	cmd.SetArgs([]string{"env-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	if strings.Contains(out, "very-secret-password") {
		t.Fatalf("output leaked PG_PASSWORD:\n%s", out)
	}
	if !strings.Contains(out, sensitiveMask) {
		t.Fatalf("output missing mask:\n%s", out)
	}
}

func TestLogDrainsGet_MasksConfigByDefault(t *testing.T) {
	t.Parallel()

	drain := types.LogDrain{
		ID:        "drain-1",
		ProjectID: "proj-test",
		Name:      "prod-datadog",
		Type:      "datadog",
		Config:    []byte(`{"api_key":"dd-very-secret-api-key","site":"us"}`),
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/log-drains/drain-1": func(w http.ResponseWriter, r *http.Request) {
			respondJSON(t, w, http.StatusOK, drain)
		},
	})
	state := newTestState(t, srv)
	cmd := newLogDrainsGetCommand(state)
	cmd.SetArgs([]string{"drain-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	if strings.Contains(out, "dd-very-secret-api-key") {
		t.Fatalf("output leaked Datadog API key:\n%s", out)
	}
	if !strings.Contains(out, sensitiveMask) {
		t.Fatalf("output missing mask placeholder:\n%s", out)
	}
}
