package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestJobsPause_Success verifies POST /v1/jobs/{id}/pause is called and the
// response is forwarded to the caller.
func TestJobsPause_Success(t *testing.T) {
	t.Parallel()

	pauseCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs/job-1/pause": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			pauseCalled = true
			respondJSON(t, w, http.StatusOK, map[string]any{"id": "job-1", "status": "paused"})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsPauseCommand(state)
	cmd.SetArgs([]string{"job-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !pauseCalled {
		t.Fatal("expected POST /v1/jobs/job-1/pause to be called")
	}
	if !strings.Contains(out, "job-1") {
		t.Fatalf("expected job-1 in output, got: %s", out)
	}
}

// TestJobsPause_InvalidID verifies that a bare invalid identifier is rejected
// before making any network call.
func TestJobsPause_InvalidID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsPauseCommand(state)
	cmd.SetArgs([]string{"../traversal"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid id")
	}
}

// TestJobsBatchEnable_SendsIDs verifies POST /v1/jobs/batch-enable is called
// with the correct ids array in the request body.
func TestJobsBatchEnable_SendsIDs(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/jobs/batch-enable": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusOK, map[string]any{"enabled": 2})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsBatchEnableCommand(state)
	cmd.SetArgs([]string{"--id", "job-1", "--id", "job-2"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, ok := receivedBody["ids"].([]any)
	if !ok || len(ids) != 2 {
		t.Fatalf("expected ids=[job-1,job-2] in body, got: %v", receivedBody)
	}
	if ids[0] != "job-1" || ids[1] != "job-2" {
		t.Fatalf("unexpected ids in body: %v", ids)
	}
}

// TestJobsBatchEnable_RequiresAtLeastOneID verifies that batch-enable with no.
func TestJobsBatchEnable_RequiresAtLeastOneID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobsBatchEnableCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one --id") {
		t.Fatalf("expected at-least-one-id error, got: %v", err)
	}
}

// TestEventSourcesSubscribe_SendsBody verifies POST /v1/event-sources/{id}/subscribe
// is called with the correct target_type and target_id in the request body.
func TestEventSourcesSubscribe_SendsBody(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/event-sources/src-1/subscribe": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, map[string]any{
				"id":          "sub-1",
				"source_id":   "src-1",
				"target_type": "job",
				"target_id":   "job-1",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newEventSourcesSubscribeCommand(state)
	cmd.SetArgs([]string{"src-1", "--target-type", "job", "--target-id", "job-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if receivedBody["target_type"] != "job" {
		t.Fatalf("expected target_type=job in body, got: %v", receivedBody)
	}
	if receivedBody["target_id"] != "job-1" {
		t.Fatalf("expected target_id=job-1 in body, got: %v", receivedBody)
	}
	if !strings.Contains(out, "sub-1") {
		t.Fatalf("expected sub-1 in output, got: %s", out)
	}
}

// TestEventSourcesSubscribe_RequiresTargetType verifies that subscribe without.
func TestEventSourcesSubscribe_RequiresTargetType(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newEventSourcesSubscribeCommand(state)
	cmd.SetArgs([]string{"src-1", "--target-id", "job-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--target-type") {
		t.Fatalf("expected --target-type error, got: %v", err)
	}
}

// TestSecretsGet_Success verifies GET /v1/secrets/{id} is called and the
// response is forwarded to the caller.
func TestSecretsGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/secrets/secret-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{
				"id":   "secret-1",
				"name": "DATABASE_URL",
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newSecretsCoverageGetCommand(state)
	cmd.SetArgs([]string{"secret-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "secret-1") {
		t.Fatalf("expected secret-1 in output, got: %s", out)
	}
}

// TestAPIKeysExpiringSoon_Success verifies GET /v1/api-keys/expiring-soon is
// called and the response is forwarded to the caller.
func TestAPIKeysExpiringSoon_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/api-keys/expiring-soon": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "key-expiring", "name": "Old CI Key", "key_prefix": "sk_"},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newAPIKeysExpiringSoonCommand(state)
	cmd.SetArgs([]string{})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "key-expiring") {
		t.Fatalf("expected key-expiring in output, got: %s", out)
	}
}
