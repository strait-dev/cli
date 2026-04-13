package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

var stdioCaptureMu sync.Mutex

func TestMain(m *testing.M) {
	// Command tests still exercise process-global stdout and cwd paths.
	// Keep package-level test parallelism at 1 so race and mutation runs are stable
	// until the command output layer is fully writer-injected.
	_ = flag.Set("test.parallel", "1")
	os.Exit(m.Run())
}

// newTestServer creates an httptest.Server and registers cleanup.
func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newTestState creates an appState pointing at the given test server.
// CI mode is enabled and output format is JSON so tests never block on
// TTY prompts or styled output.
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

func captureCommandOutput(t *testing.T, fn func()) string {
	t.Helper()
	stdout, _ := captureCommandStreams(t, fn)
	return stdout
}

func captureCommandErrorOutput(t *testing.T, fn func()) string {
	t.Helper()
	_, stderr := captureCommandStreams(t, fn)
	return stderr
}

// captureCommandStreams captures everything written to os.Stdout and os.Stderr
// during fn and restores both streams before releasing the shared lock.
func captureCommandStreams(t *testing.T, fn func()) (string, string) {
	t.Helper()
	stdioCaptureMu.Lock()
	defer stdioCaptureMu.Unlock()

	origStdout := os.Stdout
	origStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW

	fn()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	stdoutData, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	stderrData, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	_ = stdoutR.Close()
	_ = stderrR.Close()
	return string(stdoutData), string(stderrData)
}

func forceStdoutTTY(t *testing.T, tty bool) {
	t.Helper()
	prev := stdoutIsTTYFunc
	stdoutIsTTYFunc = func() bool { return tty }
	t.Cleanup(func() {
		stdoutIsTTYFunc = prev
	})
}

func forceRunsTimeNow(t *testing.T, fn func() time.Time) {
	t.Helper()
	prev := runsTimeNow
	runsTimeNow = fn
	t.Cleanup(func() {
		runsTimeNow = prev
	})
}

func forceRunsAfter(t *testing.T, fn func(time.Duration) <-chan time.Time) {
	t.Helper()
	prev := runsAfter
	runsAfter = fn
	t.Cleanup(func() {
		runsAfter = prev
	})
}

func forceLogsTimeNow(t *testing.T, fn func() time.Time) {
	t.Helper()
	prev := logsTimeNow
	logsTimeNow = fn
	t.Cleanup(func() {
		logsTimeNow = prev
	})
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
