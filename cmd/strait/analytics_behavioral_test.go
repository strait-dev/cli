package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/strait-dev/cli/internal/client"
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
			if r.URL.Query().Get("from") == "" || r.URL.Query().Get("to") == "" {
				t.Errorf("expected from/to query params, got %q", r.URL.RawQuery)
			}
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
		"GET /v1/analytics/jobs/reliability": func(w http.ResponseWriter, r *http.Request) {
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

func TestAnalyticsPerformance_Success(t *testing.T) {
	t.Parallel()

	perf := client.PerformanceAnalytics{
		SlowestJobs: []client.JobPerformance{
			{JobID: "job-1", JobSlug: "slow-job", AvgDurationSecs: 12.5, P95DurationSecs: 40.0, TotalRuns: 200, FailedRuns: 5},
		},
		Throughput: client.ThroughputStats{Completed: 190, Failed: 5, PeriodHours: 168},
		HealthSummary: client.HealthSummary{
			TotalJobs: 10, ActiveJobs: 8, SuccessRate: 0.97, AvgDurationSecs: 9.1, QueueDepth: 4,
		},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/performance": func(w http.ResponseWriter, r *http.Request) {
			assertQuery(t, r, "project_id", "proj-test")
			assertQuery(t, r, "period_hours", "168")
			respondJSON(t, w, http.StatusOK, perf)
		},
	})

	state := newTestState(t, srv)
	cmd := newAnalyticsPerformanceCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--period", "7d"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "slow-job") {
		t.Fatalf("expected slowest job slug in output: %s", out)
	}
}

func TestAnalyticsTopFailing_Success(t *testing.T) {
	t.Parallel()

	items := []types.TopFailingJob{
		{JobID: "job-1", JobSlug: "flaky", TotalRuns: 100, FailedRuns: 30, FailureRate: 0.30},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/analytics/jobs/top-failing": func(w http.ResponseWriter, r *http.Request) {
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
