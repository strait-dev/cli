package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestWhoami_Authenticated(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
	})

	state := newTestState(t, srv)
	state.opts.contextName = "prod"
	state.configPath = "/tmp/test-config.yaml"
	cmd := newWhoamiCommand(state)

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "prod") {
		t.Fatalf("expected context name in output, got: %s", out)
	}
	if !strings.Contains(out, "proj-test") {
		t.Fatalf("expected project ID in output, got: %s", out)
	}
	if !strings.Contains(out, "authenticated") {
		t.Fatalf("expected authenticated field in output, got: %s", out)
	}
}

func TestWhoami_NotAuthenticated(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not call API when not authenticated")
	}))

	state := newTestState(t, srv)
	state.opts.apiKey = ""
	cmd := newWhoamiCommand(state)

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"authenticated": false`) {
		t.Fatalf("expected authenticated: false in output, got: %s", out)
	}
}
