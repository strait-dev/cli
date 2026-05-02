package main

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/strait-dev/cli/internal/types"
)

const testUUID = "11111111-2222-3333-4444-555555555555"

// TestResolveJobIdentifier_UUIDShortCircuit asserts that when the argument is
// a well-formed UUID the resolver returns it without making any API calls,
// avoiding a wasted speculative GET.
func TestResolveJobIdentifier_UUIDShortCircuit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/" + testUUID: func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	id, err := resolveJobIdentifier(t.Context(), cli, state, testUUID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != testUUID {
		t.Fatalf("got %s, want %s", id, testUUID)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected 0 API calls for UUID input, got %d", calls.Load())
	}
}

// TestResolveEnvironmentIdentifier_UUIDShortCircuit — same coverage for environments.
func TestResolveEnvironmentIdentifier_UUIDShortCircuit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/environments/" + testUUID: func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondJSON(t, w, http.StatusOK, types.Environment{ID: testUUID})
		},
		"GET /v1/environments": func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Environment{{ID: testUUID, Slug: "prod"}})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	id, err := resolveEnvironmentIdentifier(t.Context(), cli, state, testUUID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != testUUID {
		t.Fatalf("got %s, want %s", id, testUUID)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected 0 API calls for UUID input, got %d", calls.Load())
	}
}

// TestResolveWorkflowIdentifier_UUIDShortCircuit — same coverage for workflows.
func TestResolveWorkflowIdentifier_UUIDShortCircuit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/workflows/" + testUUID: func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondJSON(t, w, http.StatusOK, types.Workflow{ID: testUUID})
		},
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Workflow{{ID: testUUID, Slug: "deploy"}})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	id, err := resolveWorkflowIdentifier(t.Context(), cli, state, testUUID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != testUUID {
		t.Fatalf("got %s, want %s", id, testUUID)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected 0 API calls for UUID input, got %d", calls.Load())
	}
}

// TestResolveJobIdentifier_DoesNotFallbackOn500 asserts that when GET /v1/jobs/X
// returns a 5xx, the resolver returns the error verbatim and does NOT fall
// back to the list+slug-match path. Falling back on a transient failure can
// mask outages and produce confusing "not found" errors when the real error
// was a server outage.
func TestResolveJobIdentifier_DoesNotFallbackOn500(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusInternalServerError, "boom")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	_, err = resolveJobIdentifier(t.Context(), cli, state, "job-1")
	if err == nil {
		t.Fatal("expected error from 5xx, got nil")
	}
	// The error message must surface the real status, not a generic "not found".
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error should mention 500: %v", err)
	}
	if listCalls.Load() != 0 {
		t.Fatalf("resolver fell back to list on 500 (expected only on 404); list calls: %d", listCalls.Load())
	}
}

// TestResolveJobIdentifier_DoesNotFallbackOn403 asserts that an authorization
// failure does not get masked as a slug-not-found.
func TestResolveJobIdentifier_DoesNotFallbackOn403(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusForbidden, "forbidden")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	_, err := resolveJobIdentifier(t.Context(), cli, state, "job-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error should mention 403: %v", err)
	}
	if listCalls.Load() != 0 {
		t.Fatalf("resolver fell back to list on 403; list calls: %d", listCalls.Load())
	}
}

// TestResolveJobIdentifier_FallsBackOn404 confirms the new branch still works:
// 404 from GetJob → fall back to ListJobs and find by slug.
func TestResolveJobIdentifier_FallsBackOn404(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/test-job": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "not found")
		},
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Job{testJob})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	id, err := resolveJobIdentifier(t.Context(), cli, state, "test-job")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != testJob.ID {
		t.Fatalf("got %s, want %s", id, testJob.ID)
	}
	if listCalls.Load() != 1 {
		t.Fatalf("expected exactly 1 list call, got %d", listCalls.Load())
	}
}

// TestResolveEnvironmentIdentifier_DoesNotFallbackOn500 ensures the new
// resolver written for environments also avoids the masking bug.
func TestResolveEnvironmentIdentifier_DoesNotFallbackOn500(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/environments/env-1": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusBadGateway, "upstream broken")
		},
		"GET /v1/environments": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.Environment{{ID: "env-1", Slug: "prod"}})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	_, err := resolveEnvironmentIdentifier(t.Context(), cli, state, "env-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error should mention 502: %v", err)
	}
	if listCalls.Load() != 0 {
		t.Fatalf("resolver fell back to list on 502; list calls: %d", listCalls.Load())
	}
}

// TestResolveEventSourceIdentifier_DoesNotFallbackOn500 — same coverage for
// the new event sources resolver.
func TestResolveEventSourceIdentifier_DoesNotFallbackOn500(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/event-sources/src-1": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusServiceUnavailable, "down for maintenance")
		},
		"GET /v1/event-sources": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.EventSource{{ID: "src-1", Slug: "github"}})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	_, err := resolveEventSourceIdentifier(t.Context(), cli, state, "src-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("error should mention 503: %v", err)
	}
	if listCalls.Load() != 0 {
		t.Fatalf("resolver fell back to list on 503; list calls: %d", listCalls.Load())
	}
}

// TestResolveJobGroupIdentifier_DoesNotFallbackOn500 — same coverage for the
// new job groups resolver.
func TestResolveJobGroupIdentifier_DoesNotFallbackOn500(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusGatewayTimeout, "timeout")
		},
		"GET /v1/job-groups": func(w http.ResponseWriter, _ *http.Request) {
			listCalls.Add(1)
			respondPaginated(t, w, http.StatusOK, []types.JobGroup{{ID: "grp-1", Slug: "default"}})
		},
	})

	state := newTestState(t, srv)
	cli, _ := newAPIClient(state)

	_, err := resolveJobGroupIdentifier(t.Context(), cli, state, "grp-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "504") {
		t.Fatalf("error should mention 504: %v", err)
	}
	if listCalls.Load() != 0 {
		t.Fatalf("resolver fell back to list on 504; list calls: %d", listCalls.Load())
	}
}
