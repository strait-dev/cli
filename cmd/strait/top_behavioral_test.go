package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func decodeJSONArray(t *testing.T, out string) []map[string]any {
	t.Helper()
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("decode output: %v\noutput: %s", err, out)
	}
	return decoded
}

func TestTopQueueCommand_ValidatesIntervalAndLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "interval must be positive",
			args:    []string{"--interval", "0s"},
			wantErr: "interval must be greater than zero",
		},
		{
			name:    "limit must be positive",
			args:    []string{"--limit", "0"},
			wantErr: "limit must be greater than zero",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := &appState{opts: &rootOptions{
				serverURL:    "http://127.0.0.1",
				apiKey:       "test-key",
				outputFormat: "json",
			}}
			cmd := newTopQueueCommand(state)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestTopQueueCommand_GlobalJSON(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)
	forceTopTimeNow(t, func() time.Time { return now })

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{
				"queued":    7,
				"executing": 3,
				"delayed":   2,
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.projectID = ""
	cmd := newTopQueueCommand(state)

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	rows := decodeJSONArray(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %#v", len(rows), rows)
	}
	wantMetrics := []string{"queued", "executing", "delayed"}
	wantValues := []float64{7, 3, 2}
	for i, wantMetric := range wantMetrics {
		if rows[i]["metric"] != wantMetric || rows[i]["value"] != wantValues[i] || rows[i]["scope"] != "global" {
			t.Fatalf("unexpected row %d: %#v", i, rows[i])
		}
		if rows[i]["sampled_at"] != now.Format(time.RFC3339) {
			t.Fatalf("unexpected sampled_at in row %d: %#v", i, rows[i])
		}
	}
}

func TestTopQueueCommand_GlobalTTY(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]any{
				"queued":    4,
				"executing": 2,
				"delayed":   1,
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	state.opts.projectID = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newTopQueueCommand(state)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Queue", "Queued", "4", "Executing", "2", "Delayed", "1"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}

func TestTopQueueCommand_ProjectJSONUsesStateProjectAndCountsStatuses(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)
	forceTopTimeNow(t, func() time.Time { return now })

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-state")
			assertQuery(t, r, "limit", "500")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", Status: types.StatusQueued},
				{ID: "run-2", Status: types.StatusQueued},
				{ID: "run-3", Status: types.StatusExecuting},
				{ID: "run-4", Status: types.StatusDelayed},
				{ID: "run-5", Status: types.StatusWaiting},
				{ID: "run-6", Status: types.StatusFailed},
				{ID: "run-7", Status: types.StatusTimedOut},
				{ID: "run-8", Status: types.StatusCrashed},
				{ID: "run-9", Status: types.StatusSystemFailed},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.projectID = "proj-state"

	cmd := newTopQueueCommand(state)
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	rows := decodeJSONArray(t, out)
	want := map[string]float64{
		"queued":    2,
		"executing": 1,
		"delayed":   1,
		"waiting":   1,
		"failed":    4,
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d: %#v", len(rows), rows)
	}
	for _, row := range rows {
		metric := row["metric"].(string)
		if row["scope"] != "proj-state" || row["sampled_at"] != now.Format(time.RFC3339) || row["value"] != want[metric] {
			t.Fatalf("unexpected row for %s: %#v", metric, row)
		}
		delete(want, metric)
	}
	if len(want) != 0 {
		t.Fatalf("missing metrics: %#v", want)
	}
}

func TestTopQueueCommand_ProjectTTY(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-tty")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", Status: types.StatusQueued},
				{ID: "run-2", Status: types.StatusExecuting},
				{ID: "run-3", Status: types.StatusFailed},
				{ID: "run-4", Status: types.StatusSystemFailed},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newTopQueueCommand(state)
		cmd.SetArgs([]string{"--project", "proj-tty"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Queue (proj-tty)", "Queued", "1", "Executing", "1", "Failed", "2"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}

func TestTopJobsCommand_ValidatesInputs(t *testing.T) {
	t.Parallel()

	t.Run("interval must be positive", func(t *testing.T) {
		state := &appState{opts: &rootOptions{
			serverURL:    "http://127.0.0.1",
			apiKey:       "test-key",
			outputFormat: "json",
		}}
		cmd := newTopJobsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test", "--interval", "0s"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "interval must be greater than zero") {
			t.Fatalf("expected interval validation error, got: %v", err)
		}
	})

	t.Run("limit must be positive", func(t *testing.T) {
		state := &appState{opts: &rootOptions{
			serverURL:    "http://127.0.0.1",
			apiKey:       "test-key",
			outputFormat: "json",
		}}
		cmd := newTopJobsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test", "--limit", "0"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "limit must be greater than zero") {
			t.Fatalf("expected limit validation error, got: %v", err)
		}
	})

	t.Run("project required", func(t *testing.T) {
		state := &appState{opts: &rootOptions{
			serverURL:    "http://127.0.0.1",
			apiKey:       "test-key",
			outputFormat: "json",
		}}
		cmd := newTopJobsCommand(state)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "project ID is required") {
			t.Fatalf("expected missing project error, got: %v", err)
		}
	})
}

func TestTopJobsCommand_ReturnsListErrors(t *testing.T) {
	t.Parallel()

	t.Run("list jobs error", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusBadGateway, "jobs backend unavailable")
			},
		})

		state := newTestState(t, srv)
		cmd := newTopJobsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "request failed with status 502") {
			t.Fatalf("expected list jobs error, got: %v", err)
		}
	})

	t.Run("list runs error", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1", Name: "Alpha", Slug: "alpha", Enabled: true}})
			},
			"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusBadGateway, "runs backend unavailable")
			},
		})

		state := newTestState(t, srv)
		cmd := newTopJobsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "request failed with status 502") {
			t.Fatalf("expected list runs error, got: %v", err)
		}
	})
}

func TestTopJobsCommand_JSONSortsCountsAndLimitsRows(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)
	forceTopTimeNow(t, func() time.Time { return now })

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-jobs")
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-a", Name: "Alpha", Slug: "alpha", Enabled: true},
				{ID: "job-b", Name: "Beta", Slug: "beta", Enabled: false},
				{ID: "job-d", Name: "Delta", Slug: "delta", Enabled: true},
				{ID: "job-g", Name: "Gamma", Slug: "gamma", Enabled: true},
			})
		},
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-jobs")
			assertQuery(t, r, "limit", "500")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", JobID: "job-a", Status: types.StatusQueued},
				{ID: "run-2", JobID: "job-a", Status: types.StatusExecuting},
				{ID: "run-3", JobID: "job-b", Status: types.StatusExecuting},
				{ID: "run-4", JobID: "job-b", Status: types.StatusFailed},
				{ID: "run-5", JobID: "job-b", Status: types.StatusTimedOut},
				{ID: "run-6", JobID: "job-d", Status: types.StatusDelayed},
				{ID: "run-7", JobID: "job-g", Status: types.StatusWaiting},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newTopJobsCommand(state)
	cmd.SetArgs([]string{"--project", "proj-jobs", "--limit", "3"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	rows := decodeJSONArray(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows after limit, got %d: %#v", len(rows), rows)
	}

	if rows[0]["slug"] != "alpha" || rows[0]["active_runs"] != float64(2) || rows[0]["failed_runs"] != float64(0) || rows[0]["sampled_runs"] != float64(2) {
		t.Fatalf("unexpected first row: %#v", rows[0])
	}
	if rows[1]["slug"] != "beta" || rows[1]["active_runs"] != float64(1) || rows[1]["failed_runs"] != float64(2) || rows[1]["enabled"] != false {
		t.Fatalf("unexpected second row: %#v", rows[1])
	}
	if rows[2]["slug"] != "delta" || rows[2]["active_runs"] != float64(1) || rows[2]["failed_runs"] != float64(0) {
		t.Fatalf("unexpected third row: %#v", rows[2])
	}
	for _, row := range rows {
		if row["sampled_at"] != now.Format(time.RFC3339) {
			t.Fatalf("unexpected sampled_at: %#v", row)
		}
		if row["slug"] == "gamma" {
			t.Fatalf("expected gamma to be truncated by limit, got rows %#v", rows)
		}
	}
}

func TestTopJobsCommand_TTYPrintsLeaderboard(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-a", Name: "Alpha", Slug: "alpha", Enabled: true},
			})
		},
		"GET /v1/runs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", JobID: "job-a", Status: types.StatusExecuting},
				{ID: "run-2", JobID: "job-a", Status: types.StatusFailed},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		cmd := newTopJobsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"Job Activity", "alpha", "active=1", "failed=1", "total=2"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected %q in stderr, got: %s", want, stderr)
		}
	}
}

func TestRunTopLoop_Behavior(t *testing.T) {
	t.Parallel()

	t.Run("watch disabled renders once", func(t *testing.T) {
		var calls int
		cmd := newTopCommand(&appState{opts: &rootOptions{}})
		err := runTopLoop(cmd, false, time.Second, func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("render called %d times, want 1", calls)
		}
	})

	t.Run("render error returns immediately", func(t *testing.T) {
		wantErr := errors.New("boom")
		cmd := newTopCommand(&appState{opts: &rootOptions{}})
		err := runTopLoop(cmd, false, time.Second, func() error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("watch loops until context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		afterCalls := 0
		forceTopAfter(t, func(time.Duration) <-chan time.Time {
			afterCalls++
			ch := make(chan time.Time)
			if afterCalls == 1 {
				close(ch)
			}
			return ch
		})

		var calls int
		cmd := newTopCommand(&appState{opts: &rootOptions{}})
		cmd.SetContext(ctx)

		stderr := captureCommandErrorOutput(t, func() {
			err := runTopLoop(cmd, true, time.Millisecond, func() error {
				calls++
				if calls == 2 {
					cancel()
				}
				return nil
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context canceled, got %v", err)
			}
		})

		if calls != 2 {
			t.Fatalf("render called %d times, want 2", calls)
		}
		if !strings.Contains(stderr, "--- press Ctrl+C to stop ---") {
			t.Fatalf("expected watch prompt in stderr, got: %s", stderr)
		}
	})
}
