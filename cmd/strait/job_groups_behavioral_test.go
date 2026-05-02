package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// testJobGroupFixture returns a fresh JobGroup per call so parallel tests
// never alias the same struct value across goroutines.
func testJobGroupFixture() types.JobGroup {
	return types.JobGroup{
		ID:        "grp-1",
		ProjectID: "proj-test",
		Name:      "Nightly",
		Slug:      "nightly",
		Paused:    false,
		JobCount:  3,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestJobGroupsList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.JobGroup{testJobGroupFixture()})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "nightly") {
		t.Fatalf("expected slug in output: %s", out)
	}
}

func TestJobGroupsGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsGetCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobGroupsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--slug", "nightly"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestJobGroupsCreate_RequiresSlug(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newJobGroupsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "Nightly"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--slug is required") {
		t.Fatalf("expected --slug error, got: %v", err)
	}
}

func TestJobGroupsCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/job-groups": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			}
			readJSONBody(t, r, &got)
			if got.Name != "Nightly" || got.Slug != "nightly" {
				t.Errorf("body: got %+v", got)
			}
			respondJSON(t, w, http.StatusCreated, testJobGroupFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "Nightly", "--slug", "nightly"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsUpdate_RequiresAtLeastOneFlag(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsUpdateCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at-least-one-flag error, got: %v", err)
	}
}

func TestJobGroupsUpdate_PatchName(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"PATCH /v1/job-groups/grp-1": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				Name *string `json:"name"`
			}
			readJSONBody(t, r, &got)
			if got.Name == nil || *got.Name != "Renamed" {
				t.Errorf("name patch: got %v, want \"Renamed\"", got.Name)
			}
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsUpdateCommand(state)
	cmd.SetArgs([]string{"grp-1", "--name", "Renamed"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"DELETE /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsDeleteCommand(state)
	cmd.SetArgs([]string{"grp-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsJobs_Success(t *testing.T) {
	t.Parallel()

	job := types.Job{ID: "job-1", Name: "Nightly Sync", Slug: "nightly-sync", Enabled: true}
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"GET /v1/job-groups/grp-1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{job})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsJobsCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "nightly-sync") {
		t.Fatalf("expected job slug in output: %s", out)
	}
}

func TestJobGroupsPause_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"POST /v1/job-groups/grp-1/pause": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsPauseCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsResume_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"POST /v1/job-groups/grp-1/resume": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsResumeCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsStats_Success(t *testing.T) {
	t.Parallel()

	stats := types.JobGroupStats{
		GroupID:    "grp-1",
		JobCount:   3,
		RunsTotal:  100,
		RunsFailed: 5,
		RunsActive: 2,
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/grp-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"GET /v1/job-groups/grp-1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, stats)
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsStatsCommand(state)
	cmd.SetArgs([]string{"grp-1"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "grp-1") {
		t.Fatalf("expected group_id in output: %s", out)
	}
}
