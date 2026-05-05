package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/client"
)

func TestJobsPause_SetsEnabledFalse(t *testing.T) {
	t.Parallel()

	var capturedEnabled *bool
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"job": testJob})
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			var req client.UpdateJobRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			capturedEnabled = req.Enabled
			paused := testJob
			paused.Enabled = false
			respondJSON(t, w, http.StatusOK, paused)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsPauseCommand(state)
	cmd.SetArgs([]string{"job-1", "--yes"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedEnabled == nil || *capturedEnabled != false {
		t.Errorf("expected enabled=false in PATCH request, got: %v", capturedEnabled)
	}
}

func TestJobsPause_TTYSuccessMessage(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			paused := testJob
			paused.Enabled = false
			respondJSON(t, w, http.StatusOK, paused)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsPauseCommand(state)
		cmd.SetArgs([]string{"job-1", "--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "Paused job") || !strings.Contains(stderr, "job-1") {
		t.Fatalf("expected tty pause message, got: %s", stderr)
	}
}

func TestJobsResume_SetsEnabledTrue(t *testing.T) {
	t.Parallel()

	var capturedEnabled *bool
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{"job": testJob})
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			var req client.UpdateJobRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			capturedEnabled = req.Enabled
			active := testJob
			active.Enabled = true
			respondJSON(t, w, http.StatusOK, active)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobsResumeCommand(state)
	cmd.SetArgs([]string{"job-1", "--yes"})

	captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if capturedEnabled == nil || *capturedEnabled != true {
		t.Errorf("expected enabled=true in PATCH request, got: %v", capturedEnabled)
	}
}

func TestJobsResume_TTYSuccessMessage(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newJobsResumeCommand(state)
		cmd.SetArgs([]string{"job-1", "--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "Resumed job") || !strings.Contains(stderr, "job-1") {
		t.Fatalf("expected tty resume message, got: %s", stderr)
	}
}

func TestJobsPause_RequiresArg(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{projectID: "proj-test", apiKey: "key"}}
	cmd := newJobsPauseCommand(state)
	cmd.SetArgs([]string{})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing arg, got nil")
		}
	})
}

func TestJobsResume_RequiresArg(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{projectID: "proj-test", apiKey: "key"}}
	cmd := newJobsResumeCommand(state)
	cmd.SetArgs([]string{})

	captureStateOutput(t, state, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing arg, got nil")
		}
	})
}

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

func TestRunInteractiveJobEdit_JSONNoChanges(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	out := captureCommandOutput(t, func() {
		err := runInteractiveJobEdit(t.Context(), cli, state, "job-1", "true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"updated": false`) || !strings.Contains(out, `"reason": "no changes"`) {
		t.Fatalf("expected no-changes JSON output, got: %s", out)
	}
}

func TestRunInteractiveJobEdit_InvalidYAML(t *testing.T) {
	t.Parallel()

	editorPath := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(editorPath, []byte("#!/bin/sh\nprintf 'name: [' > \"$1\"\n"), 0o700); err != nil {
		t.Fatalf("write editor script: %v", err)
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	err = runInteractiveJobEdit(t.Context(), cli, state, "job-1", editorPath)
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("expected yaml parse error, got: %v", err)
	}
}

func TestRunInteractiveJobEdit_TTYUpdatesJob(t *testing.T) {
	t.Parallel()

	editorPath := filepath.Join(t.TempDir(), "editor.sh")
	script := `#!/bin/sh
cat > "$1" <<'EOF'
name: Updated Job
slug: updated-job
description: updated description
cron: ""
endpoint_url: https://example.com/hook
max_attempts: 5
timeout_secs: 120
run_ttl_secs: 30
enabled: true
EOF
`
	if err := os.WriteFile(editorPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write editor script: %v", err)
	}

	var patchBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs/job-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJob)
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &patchBody)
			updated := testJob
			updated.ID = "job-1"
			updated.Version = 2
			updated.Name = "Updated Job"
			respondJSON(t, w, http.StatusOK, updated)
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	stderr := captureCommandErrorOutput(t, func() {
		err := runInteractiveJobEdit(t.Context(), cli, state, "job-1", editorPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if patchBody["name"] != "Updated Job" || patchBody["slug"] != "updated-job" {
		t.Fatalf("expected updated patch body, got: %#v", patchBody)
	}
	if !strings.Contains(stderr, "Updated job") || !strings.Contains(stderr, "version 2") {
		t.Fatalf("expected tty interactive edit success, got: %s", stderr)
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
