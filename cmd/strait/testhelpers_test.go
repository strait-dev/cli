package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestServer creates an httptest.Server and registers cleanup.
func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newTestState creates an appState pointing at the given test server.
// CI mode is enabled and output format is JSON so tests never block on
// TTY prompts or styled output. The state's stdout is a fresh *bytes.Buffer
// so captureStateOutput can read command output without swapping the global
// os.Stdout under a process-wide mutex.
func newTestState(t *testing.T, srv *httptest.Server) *appState {
	t.Helper()
	return &appState{
		opts: &rootOptions{
			serverURL:    srv.URL,
			apiKey:       "test-key",
			projectID:    "proj-test",
			outputFormat: "json",
			timeout:      10 * time.Second,
			ciMode:       true,
			noColor:      true,
		},
		stdout: &bytes.Buffer{},
	}
}

// respondJSON writes v as JSON with the given status code.
func respondJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("respondJSON: %v", err)
	}
}

// respondPaginated wraps data in the paginated API envelope that the client
// expects: {"data": [...], "has_more": false}.
func respondPaginated(t *testing.T, w http.ResponseWriter, status int, data any) {
	t.Helper()
	envelope := map[string]any{
		"data":     data,
		"has_more": false,
	}
	respondJSON(t, w, status, envelope)
}

// respondError writes a JSON error response: {"error": msg}.
func respondError(t *testing.T, w http.ResponseWriter, status int, msg string) {
	t.Helper()
	respondJSON(t, w, status, map[string]string{"error": msg})
}

// assertMethod fails the test if the request method does not match want.
func assertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Fatalf("method: got %s, want %s", r.Method, want)
	}
}

// assertPath fails the test if the request path does not match want.
func assertPath(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.URL.Path != want {
		t.Fatalf("path: got %q, want %q", r.URL.Path, want)
	}
}

// assertAuth fails the test if the Authorization header is not "Bearer <key>".
func assertAuth(t *testing.T, r *http.Request, key string) {
	t.Helper()
	want := "Bearer " + key
	got := r.Header.Get("Authorization")
	if got != want {
		t.Fatalf("auth: got %q, want %q", got, want)
	}
}

// assertQuery fails the test if query parameter key does not equal want.
func assertQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	got := r.URL.Query().Get(key)
	if got != want {
		t.Fatalf("query %s: got %q, want %q", key, got, want)
	}
}

// readJSONBody reads and unmarshals the request body into dest.
func readJSONBody(t *testing.T, r *http.Request, dest any) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("unmarshal body: %v (body: %s)", err, string(body))
	}
}

// captureStateOutput runs fn and returns whatever was written to state's
// in-memory stdout buffer. The buffer is reset before fn runs so callers can
// invoke captureStateOutput repeatedly within a single test.
//
// Each appState owns its own buffer, so parallel tests do not contend on a
// shared global the way the previous os.Stdout pipe-swap helper did. That
// swap lived under a process-wide mutex which serialized every t.Parallel()
// test that captured output and occasionally leaked goroutines between
// siblings, producing intermittent CI flakes.
//
// If state.stdout is nil or not a *bytes.Buffer, this helper installs a
// fresh buffer — callers do not need to wire one through every inline
// appState construction in older tests.
func captureStateOutput(t *testing.T, state *appState, fn func()) string {
	t.Helper()
	buf, ok := state.stdout.(*bytes.Buffer)
	if !ok {
		buf = &bytes.Buffer{}
		state.stdout = buf
	}
	buf.Reset()
	fn()
	return buf.String()
}

// newRouterServer creates an httptest server that routes requests to handler
// functions based on "METHOD PATH" keys. Unmatched requests get 404.
func newRouterServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	return newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		// Try method-agnostic match.
		if h, ok := routes[r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
}

// testContextWithTimeout creates a context with a timeout and registers cleanup.
func testContextWithTimeout(t *testing.T, d time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx, cancel
}
