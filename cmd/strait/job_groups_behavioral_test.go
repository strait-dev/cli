package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/strait-dev/cli/internal/types"
)

func testJobGroupFixture() types.JobGroup {
	return types.JobGroup{
		ID:        "group-1",
		ProjectID: "proj-test",
		Name:      "Backfills",
		Slug:      "backfills",
		JobCount:  2,
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

	if !strings.Contains(out, "backfills") {
		t.Fatalf("expected group slug in output: %s", out)
	}
}

func TestJobGroupsCreate_SendsPayload(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/job-groups": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				ProjectID   string `json:"project_id"`
				Name        string `json:"name"`
				Slug        string `json:"slug"`
				Description string `json:"description"`
			}
			readJSONBody(t, r, &got)
			if got.ProjectID != "proj-test" || got.Name != "Backfills" || got.Slug != "backfills" || got.Description != "nightly" {
				t.Errorf("unexpected payload: %+v", got)
			}
			respondJSON(t, w, http.StatusCreated, testJobGroupFixture())
		},
	})

	state := newTestState(t, srv)
	cmd := newJobGroupsCreateCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--name", "Backfills", "--slug", "backfills", "--description", "nightly"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobGroupsActions(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/job-groups/group-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, testJobGroupFixture())
		},
		"PATCH /v1/job-groups/group-1": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				Name *string `json:"name"`
			}
			readJSONBody(t, r, &got)
			if got.Name == nil || *got.Name != "Batch" {
				t.Errorf("unexpected update payload: %+v", got)
			}
			group := testJobGroupFixture()
			group.Name = "Batch"
			respondJSON(t, w, http.StatusOK, group)
		},
		"GET /v1/job-groups/group-1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1", Slug: "sync"}})
		},
		"POST /v1/job-groups/group-1/pause-all": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
		"POST /v1/job-groups/group-1/resume-all": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
		"GET /v1/job-groups/group-1/stats": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, types.JobGroupStats{GroupID: "group-1", JobCount: 2, RunsTotal: 10})
		},
		"DELETE /v1/job-groups/group-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	tests := []*cobraCommand{
		{cmd: newJobGroupsGetCommand(state), args: []string{"group-1"}},
		{cmd: newJobGroupsUpdateCommand(state), args: []string{"group-1", "--name", "Batch"}},
		{cmd: newJobGroupsJobsCommand(state), args: []string{"group-1"}},
		{cmd: newJobGroupsPauseCommand(state), args: []string{"group-1"}},
		{cmd: newJobGroupsResumeCommand(state), args: []string{"group-1"}},
		{cmd: newJobGroupsStatsCommand(state), args: []string{"group-1"}},
		{cmd: newJobGroupsDeleteCommand(state), args: []string{"group-1", "--yes"}},
	}

	for _, tc := range tests {
		tc.cmd.SetArgs(tc.args)
		if err := captureAndExec(t, state, tc.cmd); err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.cmd.Use, err)
		}
	}
}

type cobraCommand struct {
	cmd  *cobra.Command
	args []string
}
