package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/types"

	"github.com/spf13/cobra"
)

func newUsageCommand(state *appState) *cobra.Command {
	var orgID string

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show billing-period usage and forecasts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUsageCurrent(cmd, state, orgID)
		},
	}
	cmd.PersistentFlags().StringVar(&orgID, "org-id", "", "organization ID")

	cmd.AddCommand(newUsageCurrentCommand(state, &orgID))
	cmd.AddCommand(newUsageHistoryCommand(state, &orgID))
	cmd.AddCommand(newUsageForecastCommand(state, &orgID))
	return cmd
}

// runUsageCurrent is the shared handler for both `strait usage` and
// `strait usage current`. Extracting it as a free function avoids the
// previous shim pattern of constructing a fresh cobra.Command just to
// reach its RunE — that built an entire command tree per invocation
// and obscured what the parent command actually did.
func runUsageCurrent(cmd *cobra.Command, state *appState, orgID string) error {
	if orgID == "" {
		return fmt.Errorf("--org-id is required")
	}
	cli, err := newAPIClient(state)
	if err != nil {
		return err
	}
	period, err := cli.GetCurrentUsage(cmd.Context(), orgID)
	if err != nil {
		return err
	}
	if isTTYRich(state) {
		renderUsagePeriod(os.Stderr, "Current Usage", period)
		return nil
	}
	return printData(state, period)
}

func newUsageCurrentCommand(state *appState, orgID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show usage for the active billing period",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUsageCurrent(cmd, state, *orgID)
		},
	}
}

func newUsageHistoryCommand(state *appState, orgID *string) *cobra.Command {
	var from string
	var to string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show usage for past billing periods",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *orgID == "" {
				return fmt.Errorf("--org-id is required")
			}
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required")
			}
			if _, err := time.Parse(time.DateOnly, from); err != nil {
				return fmt.Errorf("invalid --from date: %w", err)
			}
			if _, err := time.Parse(time.DateOnly, to); err != nil {
				return fmt.Errorf("invalid --to date: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			periods, err := cli.GetUsageHistory(cmd.Context(), *orgID, from, to)
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
	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD)")
	return cmd
}

func newUsageForecastCommand(state *appState, orgID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "forecast",
		Short: "Show projected end-of-period usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *orgID == "" {
				return fmt.Errorf("--org-id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			forecast, err := cli.GetUsageForecast(cmd.Context(), *orgID)
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

func renderUsagePeriod(w io.Writer, title string, period *types.UsagePeriod) {
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
