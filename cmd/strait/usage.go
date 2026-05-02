package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/types"

	"github.com/spf13/cobra"
)

func newUsageCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show billing-period usage and forecasts",
	}
	cmd.AddCommand(newUsageCurrentCommand(state))
	cmd.AddCommand(newUsageHistoryCommand(state))
	cmd.AddCommand(newUsageForecastCommand(state))

	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return newUsageCurrentCommand(state).RunE(c, nil)
	}
	return cmd
}

func newUsageCurrentCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show usage for the active billing period",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			period, err := cli.GetCurrentUsage(cmd.Context())
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				renderUsagePeriod(os.Stderr, "Current Usage", period)
				return nil
			}
			return printData(state, period)
		},
	}
}

func newUsageHistoryCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show usage for past billing periods",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			periods, err := cli.GetUsageHistory(cmd.Context())
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Usage History", len(periods)))
				for _, p := range periods {
					fmt.Fprintf(os.Stderr, "  %s → %s  runs=%d  cost=$%.2f\n",
						styles.TimestampFull(p.PeriodStart),
						styles.TimestampFull(p.PeriodEnd),
						p.Runs,
						p.CostUSD,
					)
				}
				return nil
			}
			return printData(state, periods)
		},
	}
}

func newUsageForecastCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "forecast",
		Short: "Show projected end-of-period usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			forecast, err := cli.GetUsageForecast(cmd.Context())
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				renderUsagePeriod(os.Stderr, "Forecast", forecast)
				return nil
			}
			return printData(state, forecast)
		},
	}
}

func renderUsagePeriod(w *os.File, title string, period *types.UsagePeriod) {
	lines := []string{
		styles.DetailLine("Period start", styles.TimestampFull(period.PeriodStart)),
		styles.DetailLine("Period end", styles.TimestampFull(period.PeriodEnd)),
		styles.DetailLine("Runs", fmt.Sprintf("%d", period.Runs)),
		styles.DetailLine("Workflow runs", fmt.Sprintf("%d", period.WorkflowRuns)),
		styles.DetailLine("Compute minutes", fmt.Sprintf("%.2f", period.ComputeMinutes)),
		styles.DetailLine("Cost (USD)", fmt.Sprintf("$%.2f", period.CostUSD)),
	}
	if period.IncludedQuotaPct > 0 {
		lines = append(lines, styles.DetailLine("Included quota", fmt.Sprintf("%.1f%%", period.IncludedQuotaPct)))
	}
	fmt.Fprint(w, styles.DetailBox(title, lines))
}
