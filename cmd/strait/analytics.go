package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newAnalyticsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Show project analytics (cost, reliability, top-failing jobs)",
	}
	cmd.AddCommand(newAnalyticsCostsCommand(state))
	cmd.AddCommand(newAnalyticsReliabilityCommand(state))
	cmd.AddCommand(newAnalyticsTopFailingCommand(state))
	return cmd
}

func newAnalyticsCostsCommand(state *appState) *cobra.Command {
	var projectID string
	var period string
	cmd := &cobra.Command{
		Use:   "costs",
		Short: "Show cost analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			hours, err := parsePerfPeriodHours(period)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			costs, err := cli.GetCostsAnalytics(cmd.Context(), pid, hours)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Period (hours)", fmt.Sprintf("%d", costs.PeriodHours)),
					styles.DetailLine("Total (USD)", fmt.Sprintf("$%.2f", costs.TotalUSD)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Costs", lines))
				if len(costs.ByJob) > 0 {
					fmt.Fprintln(os.Stderr, styles.SectionHeader("By Job", len(costs.ByJob)))
					for _, j := range costs.ByJob {
						fmt.Fprintf(os.Stderr, "  %s  runs=%d  $%.2f\n", styles.Bold.Render(j.JobSlug), j.Runs, j.USD)
					}
				}
				return nil
			}
			return printData(state, costs)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&period, "period", "7d", "analytics period (24h, 72h, 7d, 30d, 90d)")
	return cmd
}

func newAnalyticsReliabilityCommand(state *appState) *cobra.Command {
	var projectID string
	var period string
	cmd := &cobra.Command{
		Use:   "reliability",
		Short: "Show reliability analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			hours, err := parsePerfPeriodHours(period)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			rel, err := cli.GetReliabilityAnalytics(cmd.Context(), pid, hours)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Period (hours)", fmt.Sprintf("%d", rel.PeriodHours)),
					styles.DetailLine("Success rate", fmt.Sprintf("%.2f%%", rel.SuccessRate*100)),
					styles.DetailLine("Avg duration (s)", fmt.Sprintf("%.2f", rel.AvgDurationSecs)),
					styles.DetailLine("p95 duration (s)", fmt.Sprintf("%.2f", rel.P95DurationSecs)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Reliability", lines))
				return nil
			}
			return printData(state, rel)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&period, "period", "7d", "analytics period (24h, 72h, 7d, 30d, 90d)")
	return cmd
}

func newAnalyticsTopFailingCommand(state *appState) *cobra.Command {
	var projectID string
	var period string
	var limit int
	cmd := &cobra.Command{
		Use:   "top-failing",
		Short: "Show jobs with the highest failure rate",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			hours, err := parsePerfPeriodHours(period)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			items, err := cli.ListTopFailingJobs(cmd.Context(), pid, hours, limit)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Top Failing Jobs", len(items)))
				for _, j := range items {
					fmt.Fprintf(os.Stderr, "  %s  total=%d  failed=%d  rate=%.1f%%\n",
						styles.Bold.Render(j.JobSlug), j.TotalRuns, j.FailedRuns, j.FailureRate*100)
				}
				return nil
			}
			return printData(state, items)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&period, "period", "7d", "analytics period (24h, 72h, 7d, 30d, 90d)")
	cmd.Flags().IntVar(&limit, "limit", 10, "max jobs to return")
	return cmd
}
