package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// Events command tests.

func TestEvents_Success(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{
				{ID: "evt-1", RunID: "run-1", Type: types.EventLog, Level: "info", Message: "started", CreatedAt: now},
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newEventsCommand(state)
	cmd.SetArgs([]string{"--run", "run-1"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "evt-1") {
		t.Fatalf("expected evt-1 in output, got: %s", out)
	}
}

func TestEvents_RequiresRun(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newEventsCommand(state)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--run is required") {
		t.Fatalf("expected --run required error, got: %v", err)
	}
}

// Send command tests.

func TestSend_Success(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/events": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusOK, map[string]any{"sent": true})
		},
	})
	state := newTestState(t, srv)
	cmd := newSendCommand(state)
	cmd.SetArgs([]string{"user.signup", "--data", `{"user_id":"u1"}`})
	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if receivedBody["type"] != "user.signup" {
		t.Fatalf("expected type=user.signup, got: %v", receivedBody)
	}
}

func TestSend_EmptyType(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSendCommand(state)
	cmd.SetArgs([]string{""})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "event type is required") {
		t.Fatalf("expected event type error, got: %v", err)
	}
}

func TestSend_InvalidData(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSendCommand(state)
	cmd.SetArgs([]string{"test.event", "--data", "not-json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid --data") {
		t.Fatalf("expected invalid data error, got: %v", err)
	}
}

// Drain command tests.

func TestDrain_EmptyQueue(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{
				"queued": 0, "executing": 0, "delayed": 0,
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newDrainCommand(state)
	cmd.SetArgs([]string{"--timeout", "5s"})
	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDrain_Timeout(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{
				"queued": 0, "executing": 5, "delayed": 0,
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newDrainCommand(state)
	cmd.SetArgs([]string{"--timeout", "100ms", "--interval", "20ms"})
	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "drain timeout") {
			t.Fatalf("expected drain timeout error, got: %v", err)
		}
	})
}

// Listen command tests.

func TestListen_ReceivesRuns(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	var callCount atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			runs := []map[string]any{
				{"id": "run-1", "status": "completed", "attempt": 1, "triggered_by": "manual",
					"created_at": now.Format(time.RFC3339)},
			}
			respondPaginated(t, w, http.StatusOK, runs)
		},
	})
	state := newTestState(t, srv)

	cmd := newListenCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--interval", "50ms"})
	// Discard output to avoid racing on os.Stdout with other parallel tests.
	cmd.SetOut(io.Discard)

	// Listen runs forever, so we need a context timeout.
	ctx, cancel := testContextWithTimeout(t, 500*time.Millisecond)
	defer cancel()

	_ = cmd.ExecuteContext(ctx)

	if callCount.Load() == 0 {
		t.Fatal("expected at least one poll")
	}
}

// Trace command tests.

func TestTrace_Success(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "run-1", "job_id": "job-1", "status": "completed",
				"attempt": 1, "triggered_by": "manual",
				"created_at":  now.Format(time.RFC3339),
				"started_at":  now.Add(1 * time.Second).Format(time.RFC3339),
				"finished_at": now.Add(5 * time.Second).Format(time.RFC3339),
			})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "evt-1", "run_id": "run-1", "type": "log", "level": "info",
					"message": "processing", "created_at": now.Add(2 * time.Second).Format(time.RFC3339)},
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newTraceCommand(state)
	cmd.SetArgs([]string{"run-1"})

	// trace writes to stdout directly with fmt.Print, not printData.
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "run-1") {
		t.Fatalf("expected run-1 in output, got: %s", out)
	}
	if !strings.Contains(out, "Timeline") {
		t.Fatalf("expected Timeline header in output, got: %s", out)
	}
}

// Status command tests.

func TestStatus_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 3, "executing": 1, "delayed": 0})
		},
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})
	state := newTestState(t, srv)
	cmd := newStatusCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "queue") {
		t.Fatalf("expected queue stats in output, got: %s", out)
	}
}

// Debug command tests.

func TestDebugBundle_CreatesZip(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "run-1", "job_id": "job-1", "status": "failed",
				"attempt": 1, "triggered_by": "manual",
				"created_at": now.Format(time.RFC3339),
			})
		},
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "job-1", "name": "test", "slug": "test",
				"endpoint_url": "https://x.com",
				"created_at":   now.Format(time.RFC3339),
				"updated_at":   now.Format(time.RFC3339),
			})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})
	state := newTestState(t, srv)
	outputPath := t.TempDir() + "/debug.zip"
	cmd := newDebugBundleCommand(state)
	cmd.SetArgs([]string{"run-1", "--output", outputPath})
	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Verify zip was created.
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("zip not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("zip file is empty")
	}
}

// Fixtures command tests.

func TestFixturesCreate_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusCreated, map[string]any{
				"id": "job-fixture", "name": "fixture-minimal-job", "slug": "fixture-minimal-job",
				"endpoint_url": "http://localhost:3000/webhook",
				"created_at":   "2026-03-20T10:00:00Z", "updated_at": "2026-03-20T10:00:00Z",
			})
		},
		"POST /v1/jobs/job-fixture/trigger": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id": "run-fixture", "status": "queued",
			})
		},
	})
	state := newTestState(t, srv)
	cmd := newFixturesCreateCommand(state)
	cmd.SetArgs([]string{"--template", "minimal"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "run-fixture") {
		t.Fatalf("expected run-fixture in output, got: %s", out)
	}
}

// Open command tests.

func TestOpen_NoServer(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{serverURL: "", outputFormat: "json", ciMode: true}}
	cmd := newOpenCommand(state)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "server URL is required") {
		t.Fatalf("expected server URL error, got: %v", err)
	}
}

// API command tests.

func TestAPICommand_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/custom-endpoint": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{"result": "ok"})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPICommand(state)
	cmd.SetArgs([]string{"GET", "/v1/custom-endpoint"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected ok in output, got: %s", out)
	}
}

func TestAPICommand_WithFields(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/test": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusOK, map[string]any{"created": true})
		},
	})
	state := newTestState(t, srv)
	cmd := newAPICommand(state)
	cmd.SetArgs([]string{"POST", "/v1/test", "--field", "name=test", "--field", "count=5"})
	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if receivedBody["name"] != "test" {
		t.Fatalf("expected name=test, got: %v", receivedBody)
	}
}

func TestAPICommand_RejectsAbsoluteURL(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAPICommand(state)
	cmd.SetArgs([]string{"GET", "https://evil.com/v1/jobs"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "absolute URLs are not allowed") {
		t.Fatalf("expected absolute URL error, got: %v", err)
	}
}

func TestFieldsToJSON_MultipleTypes(t *testing.T) {
	t.Parallel()
	fields := []string{"name=test", "count=42", "active=true", "tags={\"env\":\"prod\"}"}
	got, err := fieldsToJSON(fields)
	if err != nil {
		t.Fatalf("fieldsToJSON: %v", err)
	}
	if got["name"] != "test" {
		t.Fatalf("expected string, got: %v", got["name"])
	}
	if got["count"].(float64) != 42 {
		t.Fatalf("expected number, got: %v", got["count"])
	}
	if got["active"] != true {
		t.Fatalf("expected bool, got: %v", got["active"])
	}
	tagsMap, ok := got["tags"].(map[string]any)
	if !ok || tagsMap["env"] != "prod" {
		t.Fatalf("expected object, got: %v", got["tags"])
	}
}

func TestFieldsToJSON_BadFormat(t *testing.T) {
	t.Parallel()
	_, err := fieldsToJSON([]string{"noequals"})
	if err == nil || !strings.Contains(err.Error(), "expected key=value") {
		t.Fatalf("expected format error, got: %v", err)
	}
}

func TestFieldsToJSON_EmptyKey(t *testing.T) {
	t.Parallel()
	_, err := fieldsToJSON([]string{"=value"})
	if err == nil || !strings.Contains(err.Error(), "key cannot be empty") {
		t.Fatalf("expected empty key error, got: %v", err)
	}
}

func TestFieldsToJSON_Nil(t *testing.T) {
	t.Parallel()
	got, err := fieldsToJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty fields, got: %v", got)
	}
}

// Doctor command tests.

func TestDoctorCommand_HasFlags(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{outputFormat: "json"}}
	cmd := newDoctorCommand(state)
	for _, name := range []string{"verbose", "fix", "check-endpoints", "check-manifests"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("doctor missing --%s flag", name)
		}
	}
	// --json was removed; use global --format json instead.
	if cmd.Flags().Lookup("json") != nil {
		t.Error("doctor should not have --json flag (removed in favour of --format json)")
	}
}

func TestDoctorCommand_RunsAndProducesOutput(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /health/ready": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		},
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]int{"queued": 0, "executing": 0, "delayed": 0})
		},
	})
	// Use root command so --format is a known persistent flag.
	root := newRootCommand()
	root.SetArgs([]string{"--format", "json", "--server", srv.URL, "--api-key", "test-key", "--project", "proj-1", "doctor"})
	out := captureCommandOutput(t, func() {
		// Doctor may fail due to env var checks, that's OK.
		_ = root.Execute()
	})
	// Should produce JSON array of check results.
	var checks []map[string]any
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		t.Fatalf("expected JSON array output, got: %s", out)
	}
	if len(checks) == 0 {
		t.Fatal("expected at least one check result")
	}
}
