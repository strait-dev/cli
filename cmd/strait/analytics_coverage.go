package main

import (
	"github.com/spf13/cobra"
)

// registerAnalyticsCoverageCommands adds the cloud analytics surface to the
// analytics parent command. Subcommands are grouped by area:
//
//	approvals, cost-insights
//	cost-trends, cost-top, cost-by-trigger, cost-forecast   (flat, avoids clash with existing "costs" leaf)
//	runs: timeline, duration-distribution, failure-reasons, summary, by-trigger
//	jobs: comparison, by-version, cost-ranking, history <jobID>
//	tags: summary, top-failing, cost
//	webhooks: delivery-stats, endpoint-health, top-failing
//	workflows: completion-rates, summary, step-durations <workflowID>
//	events: volume, latency
func registerAnalyticsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newAnalyticsApprovalsCommand(state))
	parent.AddCommand(newAnalyticsCostInsightsCommand(state))

	// Flat cost sub-commands — the existing "costs" leaf already occupies that
	// name so we use prefixed names to avoid a duplicate-command conflict.
	parent.AddCommand(newAnalyticsCostTrendsCommand(state))
	parent.AddCommand(newAnalyticsCostTopCommand(state))
	parent.AddCommand(newAnalyticsCostByTriggerCommand(state))
	parent.AddCommand(newAnalyticsCostForecastCommand(state))

	parent.AddCommand(newAnalyticsRunsGroupCommand(state))
	parent.AddCommand(newAnalyticsJobsGroupCommand(state))
	parent.AddCommand(newAnalyticsTagsGroupCommand(state))
	parent.AddCommand(newAnalyticsWebhooksGroupCommand(state))
	parent.AddCommand(newAnalyticsWorkflowsGroupCommand(state))
	parent.AddCommand(newAnalyticsEventsGroupCommand(state))
}

// addWindowFlags attaches --from and --to flags to cmd and binds them to the
// provided string pointers.
func addWindowFlags(cmd *cobra.Command, from, to *string) {
	cmd.Flags().StringVar(from, "from", "", "window start (RFC3339, e.g. 2026-01-01T00:00:00Z)")
	cmd.Flags().StringVar(to, "to", "", "window end (RFC3339)")
}

func newAnalyticsApprovalsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "approvals",
		Short: "Show approval analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsApprovals(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsCostInsightsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-insights",
		Short: "Show cost insight analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsCostInsights(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsCostTrendsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-trends",
		Short: "Show cost trend analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsCostsTrends(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsCostTopCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-top",
		Short: "Show top-cost analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsCostsTop(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsCostByTriggerCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-by-trigger",
		Short: "Show cost-by-trigger analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsCostsByTrigger(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsCostForecastCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-forecast",
		Short: "Show cost forecast analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsCostsForecast(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsRunsGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Run analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsRunsTimelineCommand(state))
	cmd.AddCommand(newAnalyticsRunsDurationDistributionCommand(state))
	cmd.AddCommand(newAnalyticsRunsFailureReasonsCommand(state))
	cmd.AddCommand(newAnalyticsRunsSummaryCommand(state))
	cmd.AddCommand(newAnalyticsRunsByTriggerCommand(state))
	return cmd
}

func newAnalyticsRunsTimelineCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Show run timeline analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsRunsTimeline(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsRunsDurationDistributionCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "duration-distribution",
		Short: "Show run duration distribution analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsRunsDurationDistribution(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsRunsFailureReasonsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "failure-reasons",
		Short: "Show run failure reason analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsRunsFailureReasons(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsRunsSummaryCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show run summary analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsRunsSummary(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsRunsByTriggerCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "by-trigger",
		Short: "Show run-by-trigger analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsRunsByTrigger(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsJobsGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Job analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsJobsComparisonCommand(state))
	cmd.AddCommand(newAnalyticsJobsByVersionCommand(state))
	cmd.AddCommand(newAnalyticsJobsCostRankingCommand(state))
	cmd.AddCommand(newAnalyticsJobsHistoryCommand(state))
	return cmd
}

func newAnalyticsJobsComparisonCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "comparison",
		Short: "Show job comparison analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsJobsComparison(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsJobsByVersionCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "by-version",
		Short: "Show job-by-version analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsJobsByVersion(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsJobsCostRankingCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost-ranking",
		Short: "Show job cost-ranking analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsJobsCostRanking(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsJobsHistoryCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "history <job-id>",
		Short: "Show run history analytics for a specific job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsJobHistory(cmd.Context(), args[0], pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsTagsGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Tag analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsTagsSummaryCommand(state))
	cmd.AddCommand(newAnalyticsTagsTopFailingCommand(state))
	cmd.AddCommand(newAnalyticsTagsCostCommand(state))
	return cmd
}

func newAnalyticsTagsSummaryCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show tag summary analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsTagsSummary(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsTagsTopFailingCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "top-failing",
		Short: "Show top-failing tag analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsTagsTopFailing(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsTagsCostCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show tag cost analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsTagsCost(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWebhooksGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Webhook analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsWebhooksDeliveryStatsCommand(state))
	cmd.AddCommand(newAnalyticsWebhooksEndpointHealthCommand(state))
	cmd.AddCommand(newAnalyticsWebhooksTopFailingCommand(state))
	return cmd
}

func newAnalyticsWebhooksDeliveryStatsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "delivery-stats",
		Short: "Show webhook delivery stats for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWebhooksDeliveryStats(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWebhooksEndpointHealthCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "endpoint-health",
		Short: "Show webhook endpoint health analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWebhooksEndpointHealth(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWebhooksTopFailingCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "top-failing",
		Short: "Show top-failing webhook analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWebhooksTopFailing(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWorkflowsGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflows",
		Short: "Workflow analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsWorkflowsCompletionRatesCommand(state))
	cmd.AddCommand(newAnalyticsWorkflowsSummaryCommand(state))
	cmd.AddCommand(newAnalyticsWorkflowsStepDurationsCommand(state))
	return cmd
}

func newAnalyticsWorkflowsCompletionRatesCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "completion-rates",
		Short: "Show workflow completion rate analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWorkflowsCompletionRates(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWorkflowsSummaryCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show workflow summary analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWorkflowsSummary(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsWorkflowsStepDurationsCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "step-durations <workflow-id>",
		Short: "Show step duration analytics for a specific workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsWorkflowStepDurations(cmd.Context(), args[0], pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsEventsGroupCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Event analytics sub-commands",
	}
	cmd.AddCommand(newAnalyticsEventsVolumeCommand(state))
	cmd.AddCommand(newAnalyticsEventsLatencyCommand(state))
	return cmd
}

func newAnalyticsEventsVolumeCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Show event volume analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsEventsVolume(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}

func newAnalyticsEventsLatencyCommand(state *appState) *cobra.Command {
	var projectID, from, to string
	cmd := &cobra.Command{
		Use:   "latency",
		Short: "Show event latency analytics for the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetAnalyticsEventsLatency(cmd.Context(), pid, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	addWindowFlags(cmd, &from, &to)
	return cmd
}
