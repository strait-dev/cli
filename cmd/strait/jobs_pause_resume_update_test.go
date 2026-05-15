package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/client"
)

func TestJobsUpdate_FieldFlag(t *testing.T) {
	t.Parallel()

	var capturedReq client.UpdateJobRequest
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"job": testJob})
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&capturedReq); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			updated := testJob
			updated.Name = "renamed-job"
			respondJSON(t, w, http.StatusOK, updated)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsUpdateCommand(state)
	cmd.SetArgs([]string{"job-1", "--field", "name=renamed-job"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedReq.Name == nil || *capturedReq.Name != "renamed-job" {
		t.Errorf("expected name=renamed-job in request, got: %v", capturedReq.Name)
	}
	if !strings.Contains(out, "renamed-job") {
		t.Errorf("expected renamed-job in output, got: %s", out)
	}
}

func TestJobsUpdate_MultipleFields(t *testing.T) {
	t.Parallel()

	var capturedReq client.UpdateJobRequest
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"job": testJob})
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&capturedReq); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsUpdateCommand(state)
	cmd.SetArgs([]string{"job-1", "--field", "timeout_secs=120", "--field", "max_attempts=5"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedReq.TimeoutSecs == nil || *capturedReq.TimeoutSecs != 120 {
		t.Errorf("expected timeout_secs=120, got: %v", capturedReq.TimeoutSecs)
	}
	if capturedReq.MaxAttempts == nil || *capturedReq.MaxAttempts != 5 {
		t.Errorf("expected max_attempts=5, got: %v", capturedReq.MaxAttempts)
	}
}

func TestJobsUpdate_UnsupportedField(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{projectID: "proj-test", apiKey: "key"}}
	cmd := newJobsUpdateCommand(state)
	cmd.SetArgs([]string{"job-1", "--field", "nonexistent=value"})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "unsupported field") {
			t.Fatalf("expected unsupported field error, got: %v", err)
		}
	})
}

func TestJobsUpdate_InvalidFieldFormat(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{projectID: "proj-test", apiKey: "key"}}
	cmd := newJobsUpdateCommand(state)
	cmd.SetArgs([]string{"job-1", "--field", "nameonly"})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "key=value") {
			t.Fatalf("expected key=value format error, got: %v", err)
		}
	})
}

func TestJobsUpdate_FromStdinValidationAndTTYSuccess(t *testing.T) {
	t.Parallel()

	t.Run("invalid stdin json", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, testJob)
			},
		})

		state := newTestState(t, srv)
		withMockStdin(t, "{", func() {
			cmd := newJobsUpdateCommand(state)
			cmd.SetArgs([]string{"job-1", "--stdin"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "read stdin") {
				t.Fatalf("expected stdin read error, got: %v", err)
			}
		})
	})

	t.Run("unsupported stdin patch field", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, testJob)
			},
		})

		state := newTestState(t, srv)
		withMockStdin(t, `{"unsupported":"value"}`, func() {
			cmd := newJobsUpdateCommand(state)
			cmd.SetArgs([]string{"job-1", "--stdin"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "unsupported field") {
				t.Fatalf("expected unsupported field error, got: %v", err)
			}
		})
	})

	t.Run("tty success", func(t *testing.T) {
		var capturedReq client.UpdateJobRequest
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, testJob)
			},
			"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&capturedReq); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				updated := testJob
				updated.Version = 2
				respondJSON(t, w, http.StatusOK, updated)
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd := newJobsUpdateCommand(state)
			cmd.SetArgs([]string{"job-1", "--field", "timeout_secs=120"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if capturedReq.TimeoutSecs == nil || *capturedReq.TimeoutSecs != 120 {
			t.Fatalf("expected timeout_secs patch, got: %#v", capturedReq)
		}
		if !strings.Contains(stderr, "Updated job") || !strings.Contains(stderr, "version 2") {
			t.Fatalf("expected tty update success, got: %s", stderr)
		}
	})
}

func TestApplyJobField_AllSupportedKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key string
		val string
	}{
		{"name", "new-name"},
		{"slug", "new-slug"},
		{"description", "desc"},
		{"cron", "0 * * * *"},
		{"endpoint_url", "http://example.com"},
		{"enabled", "false"},
		{"max_attempts", "3"},
		{"timeout_secs", "60"},
		{"run_ttl_secs", "3600"},
	}

	for _, tc := range cases {
		upd := &client.UpdateJobRequest{}
		if err := applyJobField(upd, tc.key, tc.val); err != nil {
			t.Errorf("applyJobField(%q, %q) returned error: %v", tc.key, tc.val, err)
		}
	}
}

func TestApplyJobField_InvalidBool(t *testing.T) {
	t.Parallel()

	upd := &client.UpdateJobRequest{}
	if err := applyJobField(upd, "enabled", "notabool"); err == nil {
		t.Error("expected error for invalid bool, got nil")
	}
}

func TestApplyJobField_InvalidInt(t *testing.T) {
	t.Parallel()

	upd := &client.UpdateJobRequest{}
	for _, key := range []string{"max_attempts", "timeout_secs", "run_ttl_secs"} {
		if err := applyJobField(upd, key, "notanint"); err == nil {
			t.Errorf("expected error for invalid int on field %q, got nil", key)
		}
	}
}

func TestApplyJobPatch_UnsupportedField(t *testing.T) {
	t.Parallel()

	upd := &client.UpdateJobRequest{}
	err := applyJobPatch(upd, map[string]any{"unsupported": "value"})
	if err == nil || !strings.Contains(err.Error(), "unsupported field") {
		t.Fatalf("expected unsupported field error, got: %v", err)
	}
}
