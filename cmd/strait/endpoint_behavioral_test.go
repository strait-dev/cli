package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
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

	// Endpoint server that validates the HMAC and responds completed.
	endpointSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("X-Strait-Timestamp")
		sig := r.Header.Get("X-Strait-Signature")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if sig != expected {
			http.Error(w, "bad signature", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Strait-Verify") != "1" {
			http.Error(w, "missing verify header", http.StatusBadRequest)
			return
		}
		var payload struct {
			JobSlug string `json:"job_slug"`
			RunID   string `json:"run_id"`
		}
		_ = json.Unmarshal(body, &payload)
		if payload.JobSlug != "process" {
			http.Error(w, "wrong slug", http.StatusBadRequest)
			return
		}
		respondJSON(t, w, http.StatusOK, map[string]string{"status": "completed", "run_id": payload.RunID})
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

	if !strings.Contains(out, "completed") {
		t.Fatalf("expected completed status, got: %s", out)
	}
}

func TestEndpointVerify_FailedStatusReturnsError(t *testing.T) {
	t.Parallel()

	endpointSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(t, w, http.StatusOK, map[string]string{"status": "failed", "error": "handler exploded"})
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
			t.Fatal("expected error for failed status")
		} else if !strings.Contains(err.Error(), "did not return completed") {
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
