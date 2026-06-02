package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunsCoverage_Children_Success(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"data": []map[string]any{
			{"id": "run-child-1", "job_id": "job-1", "status": "completed"},
		},
		"has_more": false,
	}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/children": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, payload)
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsChildrenCommand(state)
	cmd.SetArgs([]string{"run-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "run-child-1") {
		t.Fatalf("expected run-child-1 in output, got: %s", out)
	}
}

func TestRunsCoverage_Children_RejectsInvalidID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRunsChildrenCommand(state)
	cmd.SetArgs([]string{"../bad-id"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid run id")
	}
}

func TestRunsCoverage_Restart_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/restart": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "run-1", "status": "queued"})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsRestartCommand(state)
	cmd.SetArgs([]string{"run-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunsCoverage_Pause_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/pause": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "run-1", "status": "paused"})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsPauseCommand(state)
	cmd.SetArgs([]string{"run-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunsCoverage_Debug_SendsDebugMode(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/debug": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &body)
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "run-1", "debug_mode": true})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsDebugCommand(state)
	cmd.SetArgs([]string{"run-1", "--enable=true"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["debug_mode"] != true {
		t.Fatalf("expected debug_mode=true in request body, got: %v", body)
	}
}

func TestRunsCoverage_Debug_DisableFlag(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/run-1/debug": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &body)
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "run-1", "debug_mode": false})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsDebugCommand(state)
	cmd.SetArgs([]string{"run-1", "--enable=false"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["debug_mode"] != false {
		t.Fatalf("expected debug_mode=false in request body, got: %v", body)
	}
}

func TestRunsCoverage_BulkCancel_SendsRunIDs(t *testing.T) {
	t.Parallel()

	var body map[string]json.RawMessage
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/bulk-cancel": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &body)
			respondJSON(t, w, http.StatusOK, map[string]any{
				"canceled": 2,
				"total":    2,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsBulkCancelCommand(state)
	cmd.SetArgs([]string{"--id", "run-1", "--id", "run-2"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var ids []string
	if err := json.Unmarshal(body["run_ids"], &ids); err != nil {
		t.Fatalf("could not parse run_ids from body: %v (body: %v)", err, body)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 run_ids in body, got %d: %v", len(ids), ids)
	}
	if ids[0] != "run-1" || ids[1] != "run-2" {
		t.Fatalf("unexpected run_ids: %v", ids)
	}
	_ = out
}

func TestRunsCoverage_BulkCancel_RequiresID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRunsBulkCancelCommand(state)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when no --id provided")
	}
}

func TestRunsCoverage_BulkDLQReplay_SendsPayload(t *testing.T) {
	t.Parallel()

	var body map[string]json.RawMessage
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/runs/bulk-dlq-replay": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &body)
			respondJSON(t, w, http.StatusOK, map[string]any{"replayed": 1})
		},
	})

	state := newTestState(t, srv)
	cmd := newRunsBulkDLQReplayCommand(state)
	cmd.SetArgs([]string{"--id", "run-dlq-1", "--project", "proj-test", "--limit", "5"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ids []string
	if err := json.Unmarshal(body["run_ids"], &ids); err != nil {
		t.Fatalf("could not parse run_ids from body: %v", err)
	}
	if len(ids) != 1 || ids[0] != "run-dlq-1" {
		t.Fatalf("unexpected run_ids: %v", ids)
	}

	var projectID string
	if err := json.Unmarshal(body["project_id"], &projectID); err != nil {
		t.Fatalf("could not parse project_id: %v", err)
	}
	if projectID != "proj-test" {
		t.Fatalf("expected project_id=proj-test, got %q", projectID)
	}

	var limit float64
	if err := json.Unmarshal(body["limit"], &limit); err != nil {
		t.Fatalf("could not parse limit: %v", err)
	}
	if limit != 5 {
		t.Fatalf("expected limit=5, got %v", limit)
	}
}

func TestRunsCoverage_BulkDLQReplay_RequiresID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRunsBulkDLQReplayCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when no --id provided")
	}
}
