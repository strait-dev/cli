package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// parseTestURL is a thin helper so TestJoinPath_RoundTripThroughURL doesn't
// need to import net/url at the test-case body level.
func parseTestURL(s string) (*url.URL, error) { return url.Parse(s) }

func TestValidatePathSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple slug", input: "my-job", wantErr: false},
		{name: "uuid", input: "550e8400-e29b-41d4-a716-446655440000", wantErr: false},
		{name: "alphanumeric", input: "abc123", wantErr: false},
		{name: "underscores", input: "job_name", wantErr: false},
		{name: "single dot inside", input: "a.b", wantErr: false},
		{name: "trailing dot", input: "abc.", wantErr: false},

		{name: "empty", input: "", wantErr: true},
		{name: "just dot", input: ".", wantErr: true},
		{name: "dotdot", input: "..", wantErr: true},
		{name: "forward slash", input: "a/b", wantErr: true},
		{name: "backslash", input: `a\b`, wantErr: true},
		{name: "leading slash", input: "/abc", wantErr: true},
		{name: "trailing slash", input: "abc/", wantErr: true},
		{name: "embedded traversal", input: "../etc/passwd", wantErr: true},
		{name: "dotdot deeper", input: "a/../b", wantErr: true},

		{name: "null byte", input: "abc\x00def", wantErr: true},
		{name: "newline", input: "abc\ndef", wantErr: true},
		{name: "carriage return", input: "abc\rdef", wantErr: true},
		{name: "tab", input: "abc\tdef", wantErr: true},
		{name: "DEL", input: "abc\x7fdef", wantErr: true},
		{name: "control char low", input: "abc\x01def", wantErr: true},

		{name: "percent prefix", input: "%2e%2e", wantErr: true},
		{name: "percent prefix solo", input: "%", wantErr: true},

		{name: "single char", input: "a", wantErr: false},
		{name: "long valid", input: strings.Repeat("a", 256), wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePathSegment(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validatePathSegment(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("validatePathSegment(%q): unexpected error: %v", tc.input, err)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		segments []string
		want     string
		wantErr  bool
	}{
		{name: "no segments", prefix: "/v1/jobs", segments: nil, want: "/v1/jobs"},
		{name: "single segment", prefix: "/v1/jobs", segments: []string{"abc"}, want: "/v1/jobs/abc"},
		{name: "multiple segments", prefix: "/v1/jobs", segments: []string{"abc", "verb"}, want: "/v1/jobs/abc/verb"},
		{name: "uuid", prefix: "/v1/runs", segments: []string{"550e8400-e29b-41d4-a716-446655440000"}, want: "/v1/runs/550e8400-e29b-41d4-a716-446655440000"},

		{name: "rejects traversal", prefix: "/v1/jobs", segments: []string{".."}, wantErr: true},
		{name: "rejects slash", prefix: "/v1/jobs", segments: []string{"a/b"}, wantErr: true},
		{name: "rejects empty", prefix: "/v1/jobs", segments: []string{""}, wantErr: true},
		{name: "rejects later segment", prefix: "/v1/jobs", segments: []string{"good", ".."}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := joinPath(tc.prefix, tc.segments...)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("joinPath: expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("joinPath: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("joinPath: got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPathTraversalRejected proves that every client method that interpolates
// a user-controlled identifier into the URL rejects path-traversal vectors at
// the call site, BEFORE making any HTTP request.
//
// If a method reaches the test server with a poisoned ID, the server fails the
// test with a clear message. The successful outcome is that every call returns
// an error and the server never sees the request.
func TestPathTraversalRejected(t *testing.T) {
	t.Parallel()

	// All known attack vectors that previously could escape the resource path.
	poisoned := []string{
		"..",
		"../etc/passwd",
		"abc/../def",
		"a/b",
		`a\b`,
		"abc\x00",
		"abc\n",
		"abc\r",
		"abc\t",
		"%2e%2e",
		"",
		".",
	}

	// Server that records any unexpected hit.
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		t.Errorf("server should never have been hit, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	ctx := context.Background()

	// Each entry runs a single client method with the poisoned id.
	// All MUST return a non-nil error and not reach the server.
	type call struct {
		name string
		fn   func(id string) error
	}

	calls := []call{
		// Jobs
		{name: "GetJob", fn: func(id string) error { _, e := c.GetJob(ctx, id); return e }},
		{name: "DeleteJob", fn: func(id string) error { return c.DeleteJob(ctx, id) }},
		{name: "UpdateJob", fn: func(id string) error { _, e := c.UpdateJob(ctx, id, UpdateJobRequest{}); return e }},
		{name: "TriggerJob", fn: func(id string) error {
			_, e := c.TriggerJob(ctx, id, TriggerJobRequest{}, "")
			return e
		}},
		{name: "BulkTriggerJob", fn: func(id string) error {
			_, e := c.BulkTriggerJob(ctx, id, BulkTriggerRequest{})
			return e
		}},
		{name: "ListJobVersions", fn: func(id string) error { _, e := c.ListJobVersions(ctx, id); return e }},
		{name: "CloneJob", fn: func(id string) error {
			_, e := c.CloneJob(ctx, id, CloneJobRequest{})
			return e
		}},
		{name: "GetJobHealth", fn: func(id string) error { _, e := c.GetJobHealth(ctx, id); return e }},
		{name: "ListJobDependencies", fn: func(id string) error { _, e := c.ListJobDependencies(ctx, id); return e }},
		{name: "AddJobDependency", fn: func(id string) error {
			_, e := c.AddJobDependency(ctx, id, AddJobDependencyRequest{})
			return e
		}},

		// Runs
		{name: "GetRun", fn: func(id string) error { _, e := c.GetRun(ctx, id); return e }},
		{name: "CancelRun", fn: func(id string) error { _, e := c.CancelRun(ctx, id); return e }},
		{name: "ReplayRun", fn: func(id string) error { _, e := c.ReplayRun(ctx, id); return e }},
		{name: "ListRunEvents", fn: func(id string) error { _, e := c.ListRunEvents(ctx, id, "", ""); return e }},
		{name: "RescheduleRun", fn: func(id string) error {
			_, e := c.RescheduleRun(ctx, id, time.Now())
			return e
		}},
		{name: "ReplayDLQ", fn: func(id string) error { _, e := c.ReplayDLQ(ctx, id); return e }},
		{name: "ListRunOutputs", fn: func(id string) error { _, e := c.ListRunOutputs(ctx, id); return e }},
		{name: "ListRunToolCalls", fn: func(id string) error { _, e := c.ListRunToolCalls(ctx, id); return e }},
		{name: "GetRunUsage", fn: func(id string) error { _, e := c.GetRunUsage(ctx, id); return e }},
		{name: "ListRunCheckpoints", fn: func(id string) error { _, e := c.ListRunCheckpoints(ctx, id); return e }},

		// Workflows
		{name: "GetWorkflow", fn: func(id string) error { _, e := c.GetWorkflow(ctx, id); return e }},
		{name: "UpdateWorkflow", fn: func(id string) error {
			_, e := c.UpdateWorkflow(ctx, id, UpdateWorkflowRequest{})
			return e
		}},
		{name: "DeleteWorkflow", fn: func(id string) error { return c.DeleteWorkflow(ctx, id) }},
		{name: "TriggerWorkflow", fn: func(id string) error {
			_, e := c.TriggerWorkflow(ctx, id, TriggerWorkflowRequest{})
			return e
		}},
		{name: "CloneWorkflow", fn: func(id string) error {
			_, e := c.CloneWorkflow(ctx, id, CloneWorkflowRequest{})
			return e
		}},
		{name: "DryRunWorkflow", fn: func(id string) error {
			_, e := c.DryRunWorkflow(ctx, id, nil)
			return e
		}},
		{name: "PlanWorkflow", fn: func(id string) error { _, e := c.PlanWorkflow(ctx, id, nil); return e }},
		{name: "SimulateWorkflow", fn: func(id string) error {
			_, e := c.SimulateWorkflow(ctx, id, nil)
			return e
		}},
		{name: "ListWorkflowVersions", fn: func(id string) error {
			_, e := c.ListWorkflowVersions(ctx, id)
			return e
		}},
		{name: "DiffWorkflowVersions", fn: func(id string) error {
			_, e := c.DiffWorkflowVersions(ctx, id, 1, 2)
			return e
		}},
		{name: "GetWorkflowPolicy", fn: func(id string) error { _, e := c.GetWorkflowPolicy(ctx, id); return e }},
		{name: "SetWorkflowPolicy", fn: func(id string) error {
			_, e := c.SetWorkflowPolicy(ctx, id, nil)
			return e
		}},

		// Workflow runs
		{name: "GetWorkflowRun", fn: func(id string) error { _, e := c.GetWorkflowRun(ctx, id); return e }},
		{name: "CancelWorkflowRun", fn: func(id string) error { _, e := c.CancelWorkflowRun(ctx, id); return e }},
		{name: "ListWorkflowStepRuns", fn: func(id string) error {
			_, e := c.ListWorkflowStepRuns(ctx, id)
			return e
		}},
		{name: "PauseWorkflowRun", fn: func(id string) error { _, e := c.PauseWorkflowRun(ctx, id); return e }},
		{name: "ResumeWorkflowRun", fn: func(id string) error { _, e := c.ResumeWorkflowRun(ctx, id); return e }},
		{name: "RetryWorkflowRun", fn: func(id string) error { _, e := c.RetryWorkflowRun(ctx, id); return e }},

		// Environments
		{name: "GetEnvironment", fn: func(id string) error { _, e := c.GetEnvironment(ctx, id); return e }},
		{name: "UpdateEnvironment", fn: func(id string) error {
			_, e := c.UpdateEnvironment(ctx, id, UpdateEnvironmentRequest{})
			return e
		}},
		{name: "DeleteEnvironment", fn: func(id string) error { return c.DeleteEnvironment(ctx, id) }},
		{name: "ListEnvironmentVariables", fn: func(id string) error {
			_, e := c.ListEnvironmentVariables(ctx, id)
			return e
		}},

		// Webhooks
		{name: "GetWebhook", fn: func(id string) error { _, e := c.GetWebhook(ctx, id); return e }},
		{name: "UpdateWebhook", fn: func(id string) error {
			_, e := c.UpdateWebhook(ctx, id, UpdateWebhookRequest{})
			return e
		}},
		{name: "DeleteWebhook", fn: func(id string) error { return c.DeleteWebhook(ctx, id) }},
		{name: "ListWebhookDeliveries", fn: func(id string) error {
			_, e := c.ListWebhookDeliveries(ctx, id, 0)
			return e
		}},
		{name: "RetryWebhookDelivery", fn: func(id string) error {
			_, e := c.RetryWebhookDelivery(ctx, id)
			return e
		}},
		{name: "TestWebhook", fn: func(id string) error { _, e := c.TestWebhook(ctx, id); return e }},

		// Event sources
		{name: "GetEventSource", fn: func(id string) error { _, e := c.GetEventSource(ctx, id); return e }},
		{name: "UpdateEventSource", fn: func(id string) error {
			_, e := c.UpdateEventSource(ctx, id, UpdateEventSourceRequest{})
			return e
		}},
		{name: "DeleteEventSource", fn: func(id string) error { return c.DeleteEventSource(ctx, id) }},

		// Job groups
		{name: "GetJobGroup", fn: func(id string) error { _, e := c.GetJobGroup(ctx, id); return e }},
		{name: "UpdateJobGroup", fn: func(id string) error {
			_, e := c.UpdateJobGroup(ctx, id, UpdateJobGroupRequest{})
			return e
		}},
		{name: "DeleteJobGroup", fn: func(id string) error { return c.DeleteJobGroup(ctx, id) }},
		{name: "ListJobsInGroup", fn: func(id string) error { _, e := c.ListJobsInGroup(ctx, id); return e }},
		{name: "PauseJobGroup", fn: func(id string) error { return c.PauseJobGroup(ctx, id) }},
		{name: "ResumeJobGroup", fn: func(id string) error { return c.ResumeJobGroup(ctx, id) }},
		{name: "GetJobGroupStats", fn: func(id string) error { _, e := c.GetJobGroupStats(ctx, id); return e }},

		// Notifications
		{name: "GetNotificationChannel", fn: func(id string) error {
			_, e := c.GetNotificationChannel(ctx, id)
			return e
		}},
		{name: "UpdateNotificationChannel", fn: func(id string) error {
			_, e := c.UpdateNotificationChannel(ctx, id, UpdateNotificationChannelRequest{})
			return e
		}},
		{name: "DeleteNotificationChannel", fn: func(id string) error {
			return c.DeleteNotificationChannel(ctx, id)
		}},

		// Log drains
		{name: "GetLogDrain", fn: func(id string) error { _, e := c.GetLogDrain(ctx, id); return e }},
		{name: "UpdateLogDrain", fn: func(id string) error {
			_, e := c.UpdateLogDrain(ctx, id, UpdateLogDrainRequest{})
			return e
		}},
		{name: "DeleteLogDrain", fn: func(id string) error { return c.DeleteLogDrain(ctx, id) }},

		// Misc
		{name: "RevokeAPIKey", fn: func(id string) error { return c.RevokeAPIKey(ctx, id) }},
		{name: "RotateAPIKey", fn: func(id string) error {
			_, e := c.RotateAPIKey(ctx, id, RotateAPIKeyRequest{})
			return e
		}},
		{name: "GetEventTrigger", fn: func(id string) error { _, e := c.GetEventTrigger(ctx, id); return e }},
		{name: "SendEvent", fn: func(id string) error { _, e := c.SendEvent(ctx, id, nil); return e }},
		{name: "DeleteServerSecret", fn: func(id string) error { return c.DeleteServerSecret(ctx, id) }},
		{name: "RemoveMember", fn: func(id string) error { return c.RemoveMember(ctx, id) }},
		{name: "DeleteTeamPolicy", fn: func(id string) error { return c.DeleteTeamPolicy(ctx, id) }},
	}

	for _, p := range poisoned {
		for _, ca := range calls {
			t.Run(ca.name+"/"+sanitizeName(p), func(t *testing.T) {
				err := ca.fn(p)
				if err == nil {
					t.Fatalf("%s(%q): expected error, got nil", ca.name, p)
				}
				if !strings.Contains(err.Error(), "invalid") {
					t.Fatalf("%s(%q): expected error to mention 'invalid', got %q", ca.name, p, err.Error())
				}
			})
		}
	}

	if got := hits.Load(); got != 0 {
		t.Fatalf("server received %d unexpected hits — path-traversal validation failed", got)
	}
}

// TestPathTraversalRejected_TwoArgWorkflowSteps covers the workflow step
// methods that take BOTH a runID and a stepRef. Either being poisoned must
// reject the call before any HTTP request is made.
func TestPathTraversalRejected_TwoArgWorkflowSteps(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should never have been hit, got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	c := mustClient(t, srv.URL)
	ctx := context.Background()

	type stepCall struct {
		name string
		fn   func(runID, stepRef string) error
	}
	calls := []stepCall{
		{name: "ApproveWorkflowStep", fn: func(r, s string) error { return c.ApproveWorkflowStep(ctx, r, s) }},
		{name: "RetryWorkflowStep", fn: func(r, s string) error { return c.RetryWorkflowStep(ctx, r, s) }},
		{name: "SkipWorkflowStep", fn: func(r, s string) error { return c.SkipWorkflowStep(ctx, r, s) }},
		{name: "ForceCompleteWorkflowStep", fn: func(r, s string) error { return c.ForceCompleteWorkflowStep(ctx, r, s) }},
	}

	cases := []struct {
		name           string
		runID, stepRef string
	}{
		{name: "poisoned run id", runID: "../events", stepRef: "step-1"},
		{name: "poisoned step ref", runID: "wfr-1", stepRef: "../skip"},
		{name: "both poisoned", runID: "..", stepRef: ".."},
		{name: "control char in run id", runID: "abc\n", stepRef: "step-1"},
		{name: "control char in step ref", runID: "wfr-1", stepRef: "abc\x00"},
		{name: "empty run id", runID: "", stepRef: "step-1"},
		{name: "empty step ref", runID: "wfr-1", stepRef: ""},
	}

	for _, ca := range calls {
		for _, tc := range cases {
			t.Run(ca.name+"/"+tc.name, func(t *testing.T) {
				err := ca.fn(tc.runID, tc.stepRef)
				if err == nil {
					t.Fatalf("%s(%q,%q): expected error, got nil", ca.name, tc.runID, tc.stepRef)
				}
			})
		}
	}
}

// TestPathTraversalRejected_DeployAndStream covers streaming methods that
// aren't part of the api.go surface.
func TestPathTraversalRejected_DeployAndStream(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should never have been hit, got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	c := mustClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	noopRun := func(RunStreamMessage) error { return nil }

	t.Run("StreamRunEvents poisoned", func(t *testing.T) {
		err := c.StreamRunEvents(ctx, "../jobs", noopRun)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// TestPathTraversalErrorWrapping verifies the validation error chain so callers
// can use errors.Is / errors.As to react if needed in future.
func TestPathTraversalErrorWrapping(t *testing.T) {
	t.Parallel()

	c := mustClient(t, "http://localhost:0")
	_, err := c.GetJob(context.Background(), "..")
	if err == nil {
		t.Fatal("expected error")
	}
	// Underlying validation error is wrapped via %w; unwrapping should yield it.
	if !errors.Is(err, errors.Unwrap(err)) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid job id") {
		t.Fatalf("expected error mentioning 'invalid job id', got %q", err.Error())
	}
}

// TestJoinPath_NoPreEncoding asserts that joinPath returns decoded path
// segments (NOT percent-encoded). Pre-encoding here would cause Go's URL
// package to double-encode the "%" sign when the result is set on
// (*url.URL).Path, producing wire-level paths like "/v1/jobs/foo%2520bar"
// instead of the intended "/v1/jobs/foo%20bar".
func TestJoinPath_NoPreEncoding(t *testing.T) {
	t.Parallel()

	// All these slug bytes are LEGAL per validatePathSegment (no slashes, no
	// control chars, no "%" prefix). joinPath must NOT pre-encode them — Go's
	// URL package handles encoding when the result is used as a URL path.
	got, err := joinPath("/v1/jobs", "my-job-v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/v1/jobs/my-job-v2" {
		t.Fatalf("got %q, want %q", got, "/v1/jobs/my-job-v2")
	}

	// Segments are joined verbatim, with no percent-encoding inserted.
	got, err = joinPath("/v1/runs", "run-abc", "events")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/v1/runs/run-abc/events" {
		t.Fatalf("got %q, want %q", got, "/v1/runs/run-abc/events")
	}
	if strings.Contains(got, "%") {
		t.Fatalf("joinPath must not introduce percent-encoding: %q", got)
	}
}

// TestJoinPath_RoundTripThroughURL ensures that when joinPath's output is set
// on (*url.URL).Path, the resulting URL string survives a round trip through
// url.Parse without altering the path — i.e., we are not silently
// double-encoding or losing segment integrity.
func TestJoinPath_RoundTripThroughURL(t *testing.T) {
	t.Parallel()

	endpoint, err := joinPath("/v1/jobs", "abc-def", "trigger")
	if err != nil {
		t.Fatalf("joinPath: %v", err)
	}
	full := "https://api.example.com" + endpoint
	parsed, err := parseTestURL(full)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Path != endpoint {
		t.Fatalf("round-trip path mismatch: got %q, want %q", parsed.Path, endpoint)
	}
	if parsed.String() != full {
		t.Fatalf("round-trip string mismatch: got %q, want %q", parsed.String(), full)
	}
}

// sanitizeName returns a test-name-safe rendering of an arbitrary string.
func sanitizeName(s string) string {
	if s == "" {
		return "empty"
	}
	r := strings.NewReplacer(
		"/", "_slash_",
		`\`, "_bslash_",
		".", "_dot_",
		"\x00", "_nul_",
		"\n", "_lf_",
		"\r", "_cr_",
		"\t", "_tab_",
		" ", "_",
		"%", "_pct_",
	)
	return r.Replace(s)
}
