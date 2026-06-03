package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

func TestLogsCommand_AutoNDJSONAndSinceFilter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	forceLogsTimeNow(t, func() time.Time { return now })

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{
				{ID: "evt-1", RunID: "run-1", Type: types.EventType("log"), Level: "info", Message: "too old", CreatedAt: now.Add(-2 * time.Minute)},
				{ID: "evt-2", RunID: "run-1", Type: types.EventType("log"), Level: "warn", Message: "recent enough", CreatedAt: now.Add(-30 * time.Second)},
			})
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, false)

	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--run", "run-1", "--since", "1m"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "too old") || !strings.Contains(out, "recent enough") {
		t.Fatalf("expected only recent row in NDJSON output, got: %s", out)
	}
	if !strings.Contains(out, "\"level\":\"warn\"") {
		t.Fatalf("expected NDJSON row, got: %s", out)
	}
}

func TestLogsCommand_InvalidSince(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{
		serverURL:    "http://127.0.0.1",
		apiKey:       "test-key",
		outputFormat: "json",
	}}

	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--run", "run-1", "--since", "tomorrow"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `invalid --since duration "tomorrow"`) {
		t.Fatalf("expected invalid duration error, got: %v", err)
	}
}

func TestLogsCommand_FollowValidationErrors(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{
		serverURL:    "http://127.0.0.1",
		apiKey:       "test-key",
		outputFormat: "json",
	}}

	t.Run("group with follow", func(t *testing.T) {
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--follow", "--group", "--run", "run-1"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--group is not supported with --follow") {
			t.Fatalf("expected group/follow validation error, got: %v", err)
		}
	})

	t.Run("follow requires run", func(t *testing.T) {
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--follow"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--follow requires --run") {
			t.Fatalf("expected missing run validation error, got: %v", err)
		}
	})
}

func TestLogsCommand_FollowRejectsTerminalRun(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusCompleted})
		},
	})

	state := newTestState(t, srv)
	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--follow", "--run", "run-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already in a terminal state") {
		t.Fatalf("expected terminal run error, got: %v", err)
	}
}

func TestLogsCommand_FollowPrintsExistingAndStreamedRows(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusExecuting})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{
				{ID: "evt-1", RunID: "run-1", Type: types.EventType("log"), Level: "info", Message: "buffered line", CreatedAt: now},
			})
		},
		"GET /v1/runs/run-1/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: event\n"))
			_, _ = w.Write([]byte(`data: {"timestamp":"` + now.Add(time.Minute).Format(time.RFC3339Nano) + `","level":"error","event_type":"log","message":"live line"}` + "\n\n"))
		},
	})

	state := newTestState(t, srv)
	state.opts.outputFormat = ""
	forceStdoutTTY(t, false)

	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--follow", "--run", "run-1", "--search", "line"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"buffered line", "live line", "\"level\":\"error\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestLogsCommand_FollowReturnsEventListError(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusExecuting})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusBadGateway, "events backend unavailable")
		},
	})

	state := newTestState(t, srv)
	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--follow", "--run", "run-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "request failed (502)") {
		t.Fatalf("expected event list error, got: %v", err)
	}
}

func TestLogsCommand_JobGlobValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("job filter requires project", func(t *testing.T) {
		state := &appState{opts: &rootOptions{
			serverURL:    "http://127.0.0.1",
			apiKey:       "test-key",
			outputFormat: "json",
		}}
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--job", "billing-*"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "project ID is required") {
			t.Fatalf("expected missing project error, got: %v", err)
		}
	})

	t.Run("list jobs failure", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusBadGateway, "jobs backend unavailable")
			},
		})
		state := newTestState(t, srv)
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test", "--job", "billing-*"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "listing jobs for --job filter") {
			t.Fatalf("expected wrapped jobs list error, got: %v", err)
		}
	})

	t.Run("invalid glob", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1", Slug: "billing-import"}})
			},
		})
		state := newTestState(t, srv)
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test", "--job", "["})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), `invalid --job glob pattern "["`) {
			t.Fatalf("expected invalid glob error, got: %v", err)
		}
	})

	t.Run("no jobs matched", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1", Slug: "billing-import"}})
			},
		})
		state := newTestState(t, srv)
		cmd := newLogsCommand(state)
		cmd.SetArgs([]string{"--project", "proj-test", "--job", "ops-*"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), `no jobs matched glob pattern "ops-*"`) {
			t.Fatalf("expected no match error, got: %v", err)
		}
	})
}

func TestLogsCommand_AggregateByJobGlobGroupsAndWarns(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.Job{
				{ID: "job-1", Slug: "billing-import"},
				{ID: "job-2", Slug: "ops-cleanup"},
			})
		},
		"GET /v1/runs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "limit", "20")
			respondPaginated(t, w, http.StatusOK, []types.JobRun{
				{ID: "run-1", JobID: "job-1", ProjectID: "proj-test"},
				{ID: "run-2", JobID: "job-1", ProjectID: "proj-test"},
				{ID: "run-3", JobID: "job-2", ProjectID: "proj-test"},
			})
		},
		"GET /v1/runs/run-1/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{
				{ID: "evt-1", RunID: "run-1", Type: types.EventType("log"), Level: "info", Message: "billing ok", CreatedAt: now},
			})
		},
		"GET /v1/runs/run-2/events": func(w http.ResponseWriter, _ *http.Request) {
			respondError(t, w, http.StatusBadGateway, "events backend unavailable")
		},
		"GET /v1/runs/run-3/events": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.RunEvent{
				{ID: "evt-2", RunID: "run-3", Type: types.EventType("log"), Level: "info", Message: "ops ignored", CreatedAt: now},
			})
		},
	})

	state := newTestState(t, srv)
	forceStdoutTTY(t, false)

	cmd := newLogsCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--job", "billing-*", "--group"})

	stdout, stderr := captureCommandStreams(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "warning: failed to fetch events for run run-2") {
		t.Fatalf("expected warning for failed event fetch, got: %s", stderr)
	}
	for _, want := range []string{`"job_slug": "billing-import"`, `"total_events": 1`, `"info": 1`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in grouped output, got: %s", want, stdout)
		}
	}
	if strings.Contains(stdout, "ops-cleanup") {
		t.Fatalf("expected non-matching jobs to be filtered out, got: %s", stdout)
	}
}

func TestPrintGroupedLogs_TTYUsesUnknownLevelBucket(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: ""}}
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		err := printGroupedLogs(state, []map[string]any{
			{"job_slug": "billing-import", "message": "missing level"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "billing-import") || !strings.Contains(stderr, "unknown") {
		t.Fatalf("expected tty grouped output with unknown level bucket, got: %s", stderr)
	}
}

func TestPrintLogRows_TailKeepsOnlyLastRow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	state := &appState{opts: &rootOptions{outputFormat: ""}}

	out := captureCommandOutput(t, func() {
		err := printLogRows(state, []map[string]any{
			{"timestamp": now, "level": "info", "type": "log", "message": "first"},
			{"timestamp": now.Add(time.Second), "level": "warn", "type": "log", "message": "second"},
		}, false, "ndjson", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "first") || !strings.Contains(out, "second") {
		t.Fatalf("expected only tailed row, got: %s", out)
	}
}

func TestPrintLogRows_TTYStylesKnownLevels(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	state := &appState{opts: &rootOptions{outputFormat: ""}}
	forceStdoutTTY(t, true)

	rows := []map[string]any{
		{"timestamp": now, "level": "error", "type": "log", "message": "boom"},
	}

	stdout, _ := captureCommandStreams(t, func() {
		err := printLogRows(state, rows, false, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stdout, "boom") {
		t.Fatalf("expected tty output, got: %s", stdout)
	}
}

func TestRunStreamRow_StatusChangeIncludesError(t *testing.T) {
	t.Parallel()

	row, ok := runStreamRow("run-1", client.RunStreamMessage{
		Type:      "status_change",
		From:      "executing",
		To:        "failed",
		Error:     "worker crashed",
		Timestamp: time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
	})
	if !ok {
		t.Fatal("expected status change row")
	}
	message, _ := row["message"].(string)
	if !strings.Contains(message, "status changed from executing to failed: worker crashed") {
		t.Fatalf("unexpected status change message: %q", message)
	}
}

func TestStreamRunLogs_AppliesSearchFilter(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, line := range []string{
				`{"timestamp":"2026-04-13T09:00:00Z","level":"info","event_type":"log","message":"ignored line"}`,
				`{"timestamp":"2026-04-13T09:00:01Z","level":"info","event_type":"log","message":"matched line"}`,
			} {
				_, _ = w.Write([]byte("event: event\n"))
				_, _ = w.Write([]byte("data: " + line + "\n\n"))
			}
		},
	})

	state := newTestState(t, srv)
	forceStdoutTTY(t, false)

	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	out := captureCommandOutput(t, func() {
		err := streamRunLogs(context.Background(), cli, state, "run-1", logFilter{Level: "", EventType: "", Search: "matched", Since: time.Time{}}, "ndjson")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "ignored line") || !strings.Contains(out, "matched line") {
		t.Fatalf("expected stream search filter to keep only matched rows, got: %s", out)
	}
}

func TestRenderFollowLogRow_NonTTYFormatsTabSeparatedOutput(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: ""}}
	forceStdoutTTY(t, false)

	row := map[string]any{
		"timestamp": time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
		"level":     "warn",
		"type":      "log",
		"message":   "plain follow row",
	}

	out := captureCommandOutput(t, func() {
		err := renderFollowLogRow(state, "", row)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "2026-04-13T09:00:00Z\twarn\tlog\tplain follow row") {
		t.Fatalf("expected plain follow output, got: %s", out)
	}
}

func TestPrintGroupedLogs_JSONSummary(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: "json"}}

	out := captureCommandOutput(t, func() {
		err := printGroupedLogs(state, []map[string]any{
			{"job_slug": "billing-import", "level": "info", "message": "one"},
			{"job_slug": "billing-import", "level": "error", "message": "two"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"job_slug": "billing-import"`) || !strings.Contains(out, `"total_events": 2`) {
		t.Fatalf("expected grouped JSON summary, got: %s", out)
	}
}

func TestStreamRunLogs_IgnoresEmptyMessages(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, line := range []string{
				`{"timestamp":"2026-04-13T09:00:00Z"}`,
				`{"timestamp":"2026-04-13T09:00:01Z","type":"event","level":"info","event_type":"log","message":"visible"}`,
			} {
				_, _ = w.Write([]byte("event: event\n"))
				_, _ = w.Write([]byte("data: " + line + "\n\n"))
			}
		},
	})

	state := newTestState(t, srv)
	forceStdoutTTY(t, false)

	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	out := captureCommandOutput(t, func() {
		err := streamRunLogs(context.Background(), cli, state, "run-1", logFilter{Level: "", EventType: "", Search: "", Since: time.Time{}}, "ndjson")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, `"message":""`) || !strings.Contains(out, "visible") {
		t.Fatalf("expected empty stream message to be ignored, got: %s", out)
	}
}

func TestPrintLogRows_NDJSONEncodeError(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: ""}}
	err := printLogRows(state, []map[string]any{
		{"timestamp": time.Now(), "level": "info", "type": "log", "message": make(chan int)},
	}, false, "ndjson", 0)
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected ndjson encode error, got: %v", err)
	}
}

func TestPrintLogRows_UsesGroupingBranch(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: "json"}}
	out := captureCommandOutput(t, func() {
		err := printLogRows(state, []map[string]any{
			{"job_slug": "billing-import", "timestamp": time.Now(), "level": "info", "type": "log", "message": "group me"},
		}, true, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, `"billing-import"`) {
		t.Fatalf("expected grouped output, got: %s", out)
	}
}

func TestRenderFollowLogRow_JSONMode(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: "json"}}
	row := map[string]any{
		"timestamp": time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
		"level":     "info",
		"type":      "log",
		"message":   "json row",
	}

	out := captureCommandOutput(t, func() {
		err := renderFollowLogRow(state, "", row)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"message":"json row"`) {
		t.Fatalf("expected json output, got: %s", out)
	}
}

func TestRunEventRowIncludesTypeAndMessage(t *testing.T) {
	t.Parallel()

	row := runEventRow("run-1", types.RunEvent{
		Type:      types.EventType("progress"),
		Level:     "info",
		Message:   "50 percent",
		CreatedAt: time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
	})

	if row["type"] != "progress" || row["message"] != "50 percent" {
		t.Fatalf("unexpected run event row: %#v", row)
	}
}

func TestPrintGroupedLogs_TTYMultipleLevels(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: ""}}
	forceStdoutTTY(t, true)

	stderr := captureCommandErrorOutput(t, func() {
		err := printGroupedLogs(state, []map[string]any{
			{"job_slug": "billing-import", "level": "info"},
			{"job_slug": "billing-import", "level": "error"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "2 event(s)") {
		t.Fatalf("expected tty event count, got: %s", stderr)
	}
}

func TestEnsureRunStreamable_PassesForExecutingRun(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusExecuting})
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	if err := ensureRunStreamable(context.Background(), cli, "run-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderFollowLogRow_TTYFormatsStyledOutput(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{outputFormat: ""}}
	forceStdoutTTY(t, true)

	row := map[string]any{
		"timestamp": time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
		"level":     "error",
		"type":      "log",
		"message":   "styled row",
	}

	out := captureCommandOutput(t, func() {
		err := renderFollowLogRow(state, "", row)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "styled row") || !strings.Contains(out, "log") {
		t.Fatalf("expected tty follow output, got: %s", out)
	}
}

func TestStreamRunLogs_StatusChangeRowsPassThrough(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/runs/run-1/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: event\n")
			_, _ = fmt.Fprint(w, "data: "+`{"timestamp":"2026-04-13T09:00:00Z","type":"status_change","from":"queued","to":"executing"}`+"\n\n")
		},
	})

	state := newTestState(t, srv)
	cli, err := newAPIClient(state)
	if err != nil {
		t.Fatalf("newAPIClient: %v", err)
	}

	out := captureCommandOutput(t, func() {
		err := streamRunLogs(context.Background(), cli, state, "run-1", logFilter{Level: "", EventType: "", Search: "", Since: time.Time{}}, "ndjson")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "status changed from queued to executing") {
		t.Fatalf("expected status change stream row, got: %s", out)
	}
}
