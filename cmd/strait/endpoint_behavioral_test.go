package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
	"github.com/strait-dev/strait-go/serve"
)

func testJobForEndpoint() types.Job {
	return types.Job{
		ID:          "job-ep",
		ProjectID:   "proj-test",
		Name:        "Process",
		Slug:        "process",
		EndpointURL: "https://example.com/old",
		Enabled:     true,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestEndpointSet_Success(t *testing.T) {
	t.Parallel()

	job := testJobForEndpoint()
	var capturedBody client.UpdateJobRequest
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
		"PATCH /v1/jobs/process": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &capturedBody)
			updated := job
			if capturedBody.EndpointURL != nil {
				updated.EndpointURL = *capturedBody.EndpointURL
			}
			respondJSON(t, w, http.StatusOK, updated)
		},
	})

	state := newTestState(t, srv)
	cmd := newEndpointSetCommand(state)
	cmd.SetArgs([]string{"process", "https://api.example.com/strait"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedBody.EndpointURL == nil || *capturedBody.EndpointURL != "https://api.example.com/strait" {
		t.Fatalf("expected endpoint_url update, got %+v", capturedBody)
	}
	if !strings.Contains(out, "https://api.example.com/strait") {
		t.Fatalf("expected new endpoint in output: %s", out)
	}
}

func TestEndpointSet_RejectsBlockedHost(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, srv)
	cmd := newEndpointSetCommand(state)
	cmd.SetArgs([]string{"process", "https://169.254.169.254/meta"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for blocked host")
	}
	if !strings.Contains(err.Error(), "internal or metadata") {
		t.Fatalf("expected metadata block error, got: %v", err)
	}
}

func TestEndpointGet_Success(t *testing.T) {
	t.Parallel()

	job := testJobForEndpoint()
	job.EndpointURL = "https://api.example.com/strait"
	job.FallbackEndpointURL = "https://fallback.example.com/strait"

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
	})

	state := newTestState(t, srv)
	cmd := newEndpointGetCommand(state)
	cmd.SetArgs([]string{"process"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "https://api.example.com/strait") {
		t.Fatalf("expected endpoint url in output: %s", out)
	}
	if !strings.Contains(out, "https://fallback.example.com/strait") {
		t.Fatalf("expected fallback url in output: %s", out)
	}
}

func TestEndpointGet_NotFound(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/missing": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusNotFound, "job not found")
		},
	})

	state := newTestState(t, srv)
	cmd := newEndpointGetCommand(state)
	cmd.SetArgs([]string{"missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEndpointVerify_Success(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"

	// Stand in for github.com/strait-dev/strait-go/serve: validate the HMAC
	// the same way the SDK does and return 404 for the canary slug because
	// no handler is registered for it. That 404 is what `endpoint verify`
	// interprets as a successful round-trip.
	endpointSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("X-Strait-Timestamp")
		expected := serve.Sign([]byte(secret), timestamp, body)
		if r.Header.Get("X-Strait-Signature") != expected {
			http.Error(w, "signature verification failed", http.StatusUnauthorized)
			return
		}
		var payload struct {
			JobSlug string `json:"job_slug"`
		}
		_ = json.Unmarshal(body, &payload)
		if payload.JobSlug != "__strait_verify__" {
			http.Error(w, "unexpected slug", http.StatusBadRequest)
			return
		}
		respondJSON(t, w, http.StatusNotFound, map[string]any{
			"success": false,
			"error":   "no handler for job slug \"__strait_verify__\"",
		})
	}))
	t.Cleanup(endpointSrv.Close)

	job := testJobForEndpoint()
	job.EndpointURL = endpointSrv.URL
	apiSrv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
	})

	state := newTestState(t, apiSrv)
	cmd := newEndpointVerifyCommand(state)
	cmd.SetArgs([]string{"process", "--secret", secret})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "verified") {
		t.Fatalf("expected verified status, got: %s", out)
	}
}

func TestEndpointVerify_BadSecretReturnsError(t *testing.T) {
	t.Parallel()

	endpointSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "signature verification failed", http.StatusUnauthorized)
	}))
	t.Cleanup(endpointSrv.Close)

	job := testJobForEndpoint()
	job.EndpointURL = endpointSrv.URL
	apiSrv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
	})

	state := newTestState(t, apiSrv)
	cmd := newEndpointVerifyCommand(state)
	cmd.SetArgs([]string{"process", "--secret", "wrong"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected error for HMAC mismatch")
		} else if !strings.Contains(err.Error(), "verification failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestEndpointVerify_5xxReturnsError(t *testing.T) {
	t.Parallel()

	endpointSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(endpointSrv.Close)

	job := testJobForEndpoint()
	job.EndpointURL = endpointSrv.URL
	apiSrv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
	})

	state := newTestState(t, apiSrv)
	cmd := newEndpointVerifyCommand(state)
	cmd.SetArgs([]string{"process", "--secret", "any"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected error for 5xx response")
		}
	})
}

func TestEndpointVerify_MissingEndpoint(t *testing.T) {
	t.Parallel()

	job := testJobForEndpoint()
	job.EndpointURL = ""
	apiSrv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/process": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, job)
		},
	})

	state := newTestState(t, apiSrv)
	cmd := newEndpointVerifyCommand(state)
	cmd.SetArgs([]string{"process", "--secret", "any"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
	if !strings.Contains(err.Error(), "no endpoint_url") {
		t.Fatalf("expected missing endpoint error, got: %v", err)
	}
}

func TestEndpointVerify_MissingSecret(t *testing.T) {
	t.Setenv("STRAIT_SIGNING_SECRET", "")
	apiSrv := newRouterServer(t, map[string]http.HandlerFunc{})
	state := newTestState(t, apiSrv)
	cmd := newEndpointVerifyCommand(state)
	cmd.SetArgs([]string{"process"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
	if !strings.Contains(err.Error(), "signing secret") {
		t.Fatalf("expected secret error, got: %v", err)
	}
}

func TestEndpointCommand_Wiring(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	endpoint := findSubcommand(t, cmd, "endpoint")
	for _, sub := range []string{"set", "get", "verify"} {
		findSubcommand(t, endpoint, sub)
	}
}
