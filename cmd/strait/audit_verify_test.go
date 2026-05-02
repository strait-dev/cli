package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAuditVerify_Passed_Text(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/audit-events/verify")
		assertAuth(t, r, "test-key")
		assertQuery(t, r, "project_id", "proj-test")
		respondJSON(t, w, http.StatusOK, map[string]any{
			"project_id":     "proj-test",
			"valid":          true,
			"events_checked": 7,
		})
	}))

	state := newTestState(t, srv)
	cmd := newAuditVerifyCommand(state)
	cmd.SetArgs([]string{"--output", "text"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(out, "PASS") {
		t.Fatalf("expected PASS in output, got %q", out)
	}
	if !strings.Contains(out, "events checked: 7") {
		t.Fatalf("expected events checked line, got %q", out)
	}
}

func TestAuditVerify_Failed_ReturnsError_JSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/audit-events/verify")
		respondJSON(t, w, http.StatusOK, map[string]any{
			"project_id":     "proj-test",
			"valid":          false,
			"events_checked": 3,
			"broken_at_id":   "ae-bad",
			"error":          "hmac mismatch",
		})
	}))

	state := newTestState(t, srv)
	cmd := newAuditVerifyCommand(state)
	cmd.SetArgs([]string{"--output", "json"})

	var rawOut string
	err := error(nil)
	rawOut = captureStateOutput(t, state, func() {
		err = cmd.ExecuteContext(context.Background())
	})
	if err == nil {
		t.Fatal("expected error on failed chain, got nil")
	}
	if !errors.Is(err, errAuditChainBroken) {
		t.Fatalf("expected errAuditChainBroken, got %v", err)
	}

	var got auditVerifyResult
	if err := json.Unmarshal([]byte(rawOut), &got); err != nil {
		t.Fatalf("unmarshal json output: %v (raw=%q)", err, rawOut)
	}
	if got.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", got.Status)
	}
	if got.EventsChecked != 3 {
		t.Fatalf("expected 3 events, got %d", got.EventsChecked)
	}
	if got.FirstBreak == nil || got.FirstBreak.EventID != "ae-bad" || got.FirstBreak.Reason != "hmac mismatch" {
		t.Fatalf("unexpected first_break: %+v", got.FirstBreak)
	}
	if got.ProjectID != "proj-test" {
		t.Fatalf("unexpected project_id: %q", got.ProjectID)
	}
}

func TestAuditVerify_InvalidOutputFlag(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	state := newTestState(t, srv)
	cmd := newAuditVerifyCommand(state)
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs([]string{"--output", "xml"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid --output")
	}
	if !strings.Contains(err.Error(), "invalid --output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditVerify_InvalidSince(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	state := newTestState(t, srv)
	cmd := newAuditVerifyCommand(state)
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs([]string{"--since", "not-a-time"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid --since")
	}
	if !strings.Contains(err.Error(), "invalid --since") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditVerify_ExitCodeMapping(t *testing.T) {
	t.Parallel()

	if code := exitCodeFromError(errAuditChainBroken); code != ExitGeneralError {
		t.Fatalf("expected ExitGeneralError (1) for broken chain, got %d", code)
	}
}
