package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/types"
)

func TestAnalyticsCosts_Success(t *testing.T) {
	t.Parallel()

	costs := types.CostsAnalytics{
		PeriodHours: 168,
		TotalUSD:    42.50,
		ByJob: []types.CostByJob{
			{JobID: "job-1", JobSlug: "backfill", Runs: 50, USD: 12.30},
		},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/costs": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "period_hours", "168")
			respondJSON(t, w, http.StatusOK, costs)
		},
	})

	state := newTestState(t, srv)
	cmd := newAnalyticsCostsCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--period", "7d"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "backfill") {
		t.Fatalf("expected job slug in output: %s", out)
	}
}

func TestAnalyticsCosts_RejectsInvalidPeriod(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newAnalyticsCostsCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--period", "bogus"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestAnalyticsReliability_Success(t *testing.T) {
	t.Parallel()

	rel := types.ReliabilityAnalytics{
		PeriodHours:     168,
		SuccessRate:     0.95,
		AvgDurationSecs: 12.3,
		P95DurationSecs: 30.1,
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/reliability": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "period_hours", "168")
			respondJSON(t, w, http.StatusOK, rel)
		},
	})

	state := newTestState(t, srv)
	cmd := newAnalyticsReliabilityCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyticsTopFailing_Success(t *testing.T) {
	t.Parallel()

	items := []types.TopFailingJob{
		{JobID: "job-1", JobSlug: "flaky", TotalRuns: 100, FailedRuns: 30, FailureRate: 0.30},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/top-failing": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "limit", "5")
			respondPaginated(t, w, http.StatusOK, items)
		},
	})

	state := newTestState(t, srv)
	cmd := newAnalyticsTopFailingCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--limit", "5"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "flaky") {
		t.Fatalf("expected job slug in output: %s", out)
	}
}
