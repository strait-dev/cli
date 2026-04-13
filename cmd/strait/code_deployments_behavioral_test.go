package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

func forceWatchCodeDeploymentUntilTerminal(t *testing.T, fn func(context.Context, *client.Client, string, string, func(string, time.Duration)) (*client.CodeDeployment, error)) {
	t.Helper()
	prev := watchCodeDeploymentUntilTerminal
	watchCodeDeploymentUntilTerminal = fn
	t.Cleanup(func() {
		watchCodeDeploymentUntilTerminal = prev
	})
}

func testCodeDeployment() client.CodeDeployment {
	return client.CodeDeployment{
		ID:              "dep-1",
		JobID:           "job-1",
		Version:         7,
		Status:          "ready",
		Runtime:         "go",
		BuiltImageURI:   "registry.example/app:7",
		ErrorMessage:    "build failed",
		SourceSizeBytes: 1024,
		CreatedAt:       time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
}

func testDeploymentJob() types.Job {
	return types.Job{
		ID:        "job-1",
		ProjectID: "proj-test",
		Slug:      "my-job",
	}
}

func TestCodeDeploymentsList_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("requires job", func(t *testing.T) {
		state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("unexpected server call")
		})))
		cmd := newCodeDeploymentsListCommand(state)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--job is required") {
			t.Fatalf("expected missing job error, got: %v", err)
		}
	})

	t.Run("lookup error", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusNotFound, "not found")
			},
		})

		state := newTestState(t, srv)
		cmd := newCodeDeploymentsListCommand(state)
		cmd.SetArgs([]string{"--job", "missing", "--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "look up job") {
			t.Fatalf("expected job lookup error, got: %v", err)
		}
	})

	t.Run("tty success", func(t *testing.T) {
		deployment := testCodeDeployment()
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
				assertQuery(t, r, "project_id", "proj-test")
				assertQuery(t, r, "slug", "my-job")
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments": func(w http.ResponseWriter, r *http.Request) {
				assertQuery(t, r, "limit", "5")
				respondPaginated(t, w, http.StatusOK, []client.CodeDeployment{deployment})
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd := newCodeDeploymentsListCommand(state)
			cmd.SetArgs([]string{"--job", "my-job", "--project", "proj-test", "--limit", "5"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{"Deployments", "dep-1", "v7", "go"} {
			if !strings.Contains(stderr, want) {
				t.Fatalf("expected %q in tty list output, got: %s", want, stderr)
			}
		}
	})
}

func TestCodeDeploymentGet_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("invalid deployment id", func(t *testing.T) {
		state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("unexpected server call")
		})))
		cmd, _ := newCodeDeploymentGetCommand(state)
		cmd.SetArgs([]string{"bad id", "--job", "my-job"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid deployment ID") {
			t.Fatalf("expected invalid id error, got: %v", err)
		}
	})

	t.Run("tty shows image and error", func(t *testing.T) {
		deployment := testCodeDeployment()
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd, _ := newCodeDeploymentGetCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{"dep-1", "Image", deployment.BuiltImageURI, "Error", deployment.ErrorMessage} {
			if !strings.Contains(stderr, want) {
				t.Fatalf("expected %q in tty get output, got: %s", want, stderr)
			}
		}
	})
}

func TestCodeDeploymentLogs_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("invalid deployment id", func(t *testing.T) {
		state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("unexpected server call")
		})))
		cmd, _ := newCodeDeploymentLogsCommand(state)
		cmd.SetArgs([]string{"bad id", "--job", "my-job"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid deployment ID") {
			t.Fatalf("expected invalid id error, got: %v", err)
		}
	})

	t.Run("prints static logs", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "ready"
		deployment.BuildLogs = "build finished\n"
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
		})

		state := newTestState(t, srv)
		out := captureCommandOutput(t, func() {
			cmd, _ := newCodeDeploymentLogsCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(out, "build finished") {
			t.Fatalf("expected build logs output, got: %s", out)
		}
	})

	t.Run("auto streams building deployment", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, r *http.Request) {
				assertQuery(t, r, "stream", "true")
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"chunk\":\"streamed log\\n\"}\n\n"))
				_, _ = w.Write([]byte("data: {\"done\":true}\n\n"))
			},
		})

		state := newTestState(t, srv)
		out := captureCommandOutput(t, func() {
			cmd, _ := newCodeDeploymentLogsCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(out, "streamed log") {
			t.Fatalf("expected streamed log output, got: %s", out)
		}
	})
}

func TestCodeDeploymentRollback_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("invalid deployment id", func(t *testing.T) {
		state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("unexpected server call")
		})))
		cmd, _ := newCodeDeploymentRollbackCommand(state)
		cmd.SetArgs([]string{"bad id", "--job", "my-job", "--yes"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid deployment ID") {
			t.Fatalf("expected invalid id error, got: %v", err)
		}
	})

	t.Run("tty success", func(t *testing.T) {
		deployment := testCodeDeployment()
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"POST /v1/jobs/job-1/deployments/dep-1/rollback": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd, _ := newCodeDeploymentRollbackCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test", "--yes"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(stderr, "Rolled back job my-job") || !strings.Contains(stderr, "v7") {
			t.Fatalf("expected rollback tty output, got: %s", stderr)
		}
	})
}

func TestCodeDeploymentWatch_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("ready tty shows image", func(t *testing.T) {
		deployment := testCodeDeployment()
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd, _ := newCodeDeploymentWatchCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(stderr, "Deployment dep-1 is ready") || !strings.Contains(stderr, "Image") {
			t.Fatalf("expected ready tty output, got: %s", stderr)
		}
	})

	t.Run("stream success then final fetch error", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		finalCalls := 0
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				finalCalls++
				if finalCalls == 1 {
					respondJSON(t, w, http.StatusOK, deployment)
					return
				}
				respondError(t, w, http.StatusInternalServerError, "fetch failed")
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"done\":true}\n\n"))
			},
		})

		state := newTestState(t, srv)
		cmd, _ := newCodeDeploymentWatchCommand(state)
		cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "fetch final status") {
			t.Fatalf("expected final fetch error, got: %v", err)
		}
	})

	t.Run("timeout returns watch timeout error", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			},
		})

		state := newTestState(t, srv)
		cmd, _ := newCodeDeploymentWatchCommand(state)
		cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test", "--timeout", "10ms"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "watch timed out after 10ms") {
			t.Fatalf("expected watch timeout error, got: %v", err)
		}
	})

	t.Run("polling ready tty output", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		final := testCodeDeployment()
		final.Status = "ready"
		forceWatchCodeDeploymentUntilTerminal(t, func(_ context.Context, _ *client.Client, jobID, deploymentID string, tick func(string, time.Duration)) (*client.CodeDeployment, error) {
			if jobID != "job-1" || deploymentID != "dep-1" {
				t.Fatalf("unexpected watch args: %s %s", jobID, deploymentID)
			}
			tick("building", 3*time.Second)
			return &final, nil
		})

		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusInternalServerError, "stream unavailable")
			},
		})

		state := newTestState(t, srv)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		stderr := captureCommandErrorOutput(t, func() {
			cmd, _ := newCodeDeploymentWatchCommand(state)
			cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{"Log stream ended", "status:", "Deployment dep-1 ready", "Image"} {
			if !strings.Contains(stderr, want) {
				t.Fatalf("expected %q in polling tty output, got: %s", want, stderr)
			}
		}
	})

	t.Run("polling error propagates", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		forceWatchCodeDeploymentUntilTerminal(t, func(_ context.Context, _ *client.Client, _ string, _ string, _ func(string, time.Duration)) (*client.CodeDeployment, error) {
			return nil, errors.New("poll failed")
		})

		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusInternalServerError, "stream unavailable")
			},
		})

		state := newTestState(t, srv)
		cmd, _ := newCodeDeploymentWatchCommand(state)
		cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "poll failed") {
			t.Fatalf("expected polling error, got: %v", err)
		}
	})

	t.Run("polling final failure uses status fallback", func(t *testing.T) {
		deployment := testCodeDeployment()
		deployment.Status = "building"
		final := testCodeDeployment()
		final.Status = "failed"
		final.ErrorMessage = ""
		forceWatchCodeDeploymentUntilTerminal(t, func(_ context.Context, _ *client.Client, _ string, _ string, _ func(string, time.Duration)) (*client.CodeDeployment, error) {
			return &final, nil
		})

		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
				respondPaginated(t, w, http.StatusOK, []types.Job{testDeploymentJob()})
			},
			"GET /v1/jobs/job-1/deployments/dep-1": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, deployment)
			},
			"GET /v1/jobs/job-1/deployments/dep-1/logs": func(w http.ResponseWriter, _ *http.Request) {
				respondError(t, w, http.StatusInternalServerError, "stream unavailable")
			},
		})

		state := newTestState(t, srv)
		cmd, _ := newCodeDeploymentWatchCommand(state)
		cmd.SetArgs([]string{"dep-1", "--job", "my-job", "--project", "proj-test"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "deployment dep-1 failed: failed") {
			t.Fatalf("expected final status fallback error, got: %v", err)
		}
	})
}
