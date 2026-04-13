package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchRunUntilDone_Completed(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			fmt.Fprint(w, `{"id":"run-1","status":"executing","attempt":1}`)
		} else {
			fmt.Fprint(w, `{"id":"run-1","status":"completed","attempt":1}`)
		}
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: "json"}}
	captureCommandOutput(t, func() {
		err := watchRunUntilDone(context.Background(), state, "run-1", 10*time.Millisecond, 5*time.Second)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
	})
}

func TestWatchRunUntilDone_Failed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"failed","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: "json"}}
	captureCommandOutput(t, func() {
		err := watchRunUntilDone(context.Background(), state, "run-1", 10*time.Millisecond, 5*time.Second)
		if err == nil {
			t.Fatal("expected error for failed run")
		}
		if !strings.Contains(err.Error(), "terminal status") {
			t.Fatalf("expected 'terminal status' in error, got: %v", err)
		}
	})
}

func TestWatchRunUntilDone_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"executing","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: "json"}}
	captureCommandOutput(t, func() {
		err := watchRunUntilDone(context.Background(), state, "run-1", 20*time.Millisecond, 100*time.Millisecond)
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "watch timeout reached") {
			t.Fatalf("expected 'watch timeout reached' in error, got: %v", err)
		}
	})
}

func TestWatchRunUntilDone_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"executing","attempt":1}`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: "json"}}
	err := watchRunUntilDone(ctx, state, "run-1", 10*time.Millisecond, 5*time.Second)
	if err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestWatchRunUntilDone_PrintsEachPoll(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			fmt.Fprint(w, `{"id":"run-1","status":"executing","attempt":1}`)
		} else {
			fmt.Fprint(w, `{"id":"run-1","status":"completed","attempt":1}`)
		}
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: "json"}}
	captureCommandOutput(t, func() {
		err := watchRunUntilDone(context.Background(), state, "run-1", 10*time.Millisecond, 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount.Load() != 3 {
			t.Fatalf("expected 3 polls, got %d", callCount.Load())
		}
	})
}

func TestRunsWatch_DefaultSucceedsOnCompleted(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"completed","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: "json"}}
	cmd := newRunsWatchCommand(state)
	cmd.SetArgs([]string{"run-1", "--interval", "10ms", "--timeout", "5s"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("expected success on completed run, got: %v", err)
		}
	})
}

func TestRunsWatch_DefaultFailsOnFailed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"failed","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: "json"}}
	cmd := newRunsWatchCommand(state)
	cmd.SetArgs([]string{"run-1", "--interval", "10ms", "--timeout", "5s"})

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error on failed run, got nil")
		}
		if !strings.Contains(err.Error(), "failed") {
			t.Errorf("expected 'failed' in error, got: %v", err)
		}
	})
}

func TestRunsWatch_UntilAcceptsFailedAsSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"failed","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: "json"}}
	cmd := newRunsWatchCommand(state)
	cmd.SetArgs([]string{"run-1", "--until", "completed,failed", "--interval", "10ms", "--timeout", "5s"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("expected success with --until completed,failed on failed run, got: %v", err)
		}
	})
}

func TestRunsWatch_UntilFailsOnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"canceled","attempt":1}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: "json"}}
	cmd := newRunsWatchCommand(state)
	cmd.SetArgs([]string{"run-1", "--until", "completed,failed", "--interval", "10ms", "--timeout", "5s"})

	captureCommandOutput(t, func() {
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error when status not in --until set, got nil")
		}
		if !strings.Contains(err.Error(), "canceled") {
			t.Errorf("expected 'canceled' in error, got: %v", err)
		}
	})
}

func TestParseUntilStatuses_EmptyReturnsNil(t *testing.T) {
	t.Parallel()

	if m := parseUntilStatuses(""); m != nil {
		t.Errorf("expected nil for empty string, got %v", m)
	}
	if m := parseUntilStatuses("  "); m != nil {
		t.Errorf("expected nil for whitespace string, got %v", m)
	}
}

func TestParseUntilStatuses_ParsesCommaSeparated(t *testing.T) {
	t.Parallel()

	m := parseUntilStatuses("completed,failed,canceled")
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	for _, want := range []string{"completed", "failed", "canceled"} {
		if !m[want] {
			t.Errorf("expected %q in parsed map, got: %v", want, m)
		}
	}
}

func TestParseUntilStatuses_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	m := parseUntilStatuses("completed , failed")
	if !m["completed"] || !m["failed"] {
		t.Errorf("expected whitespace-trimmed keys, got: %v", m)
	}
}

func TestRunsWatchCommand_HasUntilFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newRunsWatchCommand(state)
	if cmd.Flags().Lookup("until") == nil {
		t.Error("expected --until flag on runs watch command")
	}
}

func TestRunsWatch_TTYMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		status      string
		args        []string
		wantErr     string
		wantMessage string
	}{
		{
			name:        "accepted until status",
			status:      "failed",
			args:        []string{"run-1", "--until", "completed,failed", "--interval", "10ms", "--timeout", "5s"},
			wantMessage: "Run reached status failed",
		},
		{
			name:        "unexpected until status",
			status:      "canceled",
			args:        []string{"run-1", "--until", "completed,failed", "--interval", "10ms", "--timeout", "5s"},
			wantErr:     `run reached terminal status "canceled"`,
			wantMessage: "Run reached status canceled (not in --until set)",
		},
		{
			name:        "default completed",
			status:      "completed",
			args:        []string{"run-1", "--interval", "10ms", "--timeout", "5s"},
			wantMessage: "Run completed",
		},
		{
			name:        "default failed",
			status:      "failed",
			args:        []string{"run-1", "--interval", "10ms", "--timeout", "5s"},
			wantErr:     `run reached terminal status "failed"`,
			wantMessage: "Run reached terminal status failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"run-1","status":"%s","attempt":2}`, tc.status)
			}))
			defer srv.Close()

			state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: ""}}
			forceStdoutTTY(t, true)

			stderr := captureCommandErrorOutput(t, func() {
				cmd := newRunsWatchCommand(state)
				cmd.SetArgs(tc.args)

				err := cmd.Execute()
				if tc.wantErr == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})

			if !strings.Contains(stderr, tc.wantMessage) {
				t.Fatalf("expected %q in stderr, got: %s", tc.wantMessage, stderr)
			}
		})
	}
}

func TestRunsWatch_TTYTimeoutPrintsProgress(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	nowCalls := 0
	forceRunsTimeNow(t, func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return base
		}
		return base.Add(2 * time.Second)
	})
	forceRunsAfter(t, func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time)
		close(ch)
		return ch
	})
	forceStdoutTTY(t, true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"run-1","status":"executing","attempt":3}`)
	}))
	defer srv.Close()

	state := &appState{opts: &rootOptions{serverURL: srv.URL, apiKey: "test-key", outputFormat: ""}}
	stderr := captureCommandErrorOutput(t, func() {
		cmd := newRunsWatchCommand(state)
		cmd.SetArgs([]string{"run-1", "--interval", "1ms", "--timeout", "1s"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "watch timeout reached") {
			t.Fatalf("expected timeout error, got: %v", err)
		}
	})

	if !strings.Contains(stderr, "attempt=3") {
		t.Fatalf("expected progress line in stderr, got: %s", stderr)
	}
}

func TestWatchRunUntilDone_TTYMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		status      string
		wantErr     string
		wantMessage string
	}{
		{name: "completed", status: "completed", wantMessage: "Run completed"},
		{name: "failed", status: "failed", wantErr: `run reached terminal status "failed"`, wantMessage: "Run reached terminal status failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"run-1","status":"%s","attempt":1}`, tc.status)
			}))
			defer srv.Close()

			state := &appState{opts: &rootOptions{serverURL: srv.URL, outputFormat: ""}}
			forceStdoutTTY(t, true)

			stderr := captureCommandErrorOutput(t, func() {
				err := watchRunUntilDone(context.Background(), state, "run-1", 10*time.Millisecond, 5*time.Second)
				if tc.wantErr == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})

			if !strings.Contains(stderr, tc.wantMessage) {
				t.Fatalf("expected %q in stderr, got: %s", tc.wantMessage, stderr)
			}
		})
	}
}
