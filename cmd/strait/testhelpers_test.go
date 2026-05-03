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

// The helpers below run inside HTTP handler goroutines, not the test
// goroutine. Per testing.T docs, FailNow / Fatal must be called from the test
// goroutine — calling t.Fatal from a server-side goroutine only kills that
// goroutine, leaves the request hanging, and surfaces as an opaque client
// timeout instead of the real assertion. So these helpers use t.Errorf and
// then write a 5xx via http.Error so the failure surfaces synchronously
// through the client.

// respondJSON writes v as JSON with the given status code.
func respondJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("respondJSON: %v", err)
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
func assertMethod(t *testing.T, r *http.Request, w http.ResponseWriter, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("method: got %s, want %s", r.Method, want)
		http.Error(w, "method assertion failed", http.StatusInternalServerError)
	}
}

// assertPath fails the test if the request path does not match want.
func assertPath(t *testing.T, r *http.Request, w http.ResponseWriter, want string) {
	t.Helper()
	if r.URL.Path != want {
		t.Errorf("path: got %q, want %q", r.URL.Path, want)
		http.Error(w, "path assertion failed", http.StatusInternalServerError)
	}
}

// assertAuth fails the test if the Authorization header is not "Bearer <key>".
func assertAuth(t *testing.T, r *http.Request, key string) {
	t.Helper()
	want := "Bearer " + key
	got := r.Header.Get("Authorization")
	if got != want {
		t.Errorf("auth: got %q, want %q", got, want)
	}
}

// assertQuery fails the test if query parameter key does not equal want.
func assertQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	got := r.URL.Query().Get(key)
	if got != want {
		t.Errorf("query %s: got %q, want %q", key, got, want)
	}
}

// readJSONBody reads and unmarshals the request body into dest. Errors are
// recorded with t.Errorf rather than t.Fatal because this helper runs inside
// the HTTP server goroutine.
func readJSONBody(t *testing.T, r *http.Request, dest any) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read body: %v", err)
		return
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Errorf("unmarshal body: %v (body: %s)", err, string(body))
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
// If state.stdout is nil, a fresh buffer is installed. If state.stdout is
// already set to something other than *bytes.Buffer, the helper fails the
// test rather than silently replacing it — callers who inject custom writers
// should not have them swapped out from under them.
func captureStateOutput(t *testing.T, state *appState, fn func()) string {
	t.Helper()
	if state.stdout == nil {
		state.stdout = &bytes.Buffer{}
	}
	buf, ok := state.stdout.(*bytes.Buffer)
	if !ok {
		t.Fatalf("captureStateOutput: state.stdout is %T, want *bytes.Buffer", state.stdout)
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
