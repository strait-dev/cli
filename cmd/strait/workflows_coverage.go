package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

// registerWorkflowsCoverageCommands adds coverage subcommands to the workflows
// parent command. It does not duplicate any subcommands already registered by
// newWorkflowsCommand (list, get, create, update, delete, clone, versions, diff,
// dry-run, plan, simulate, trigger, visualize, policy).
func registerWorkflowsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newWorkflowsGraphCommand(state))
	parent.AddCommand(newWorkflowsActiveVersionsCommand(state))
	parent.AddCommand(newWorkflowsVersionGetCommand(state))
	parent.AddCommand(newWorkflowsVersionImpactCommand(state))
	parent.AddCommand(newWorkflowsVersionStepsCommand(state))
	parent.AddCommand(newWorkflowsCanaryGetCommand(state))
	parent.AddCommand(newWorkflowsCanarySetCommand(state))
	parent.AddCommand(newWorkflowsCanaryRollbackCommand(state))
}

func newWorkflowsGraphCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "graph <workflow-id-or-slug>",
		Short: "Get the execution graph for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowGraph(cmd.Context(), workflowID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowsActiveVersionsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "active-versions <workflow-id-or-slug>",
		Short: "Get the active versions for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowActiveVersions(cmd.Context(), workflowID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowsVersionGetCommand(state *appState) *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "version-get <workflow-id-or-slug>",
		Short: "Get a specific version of a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(versionID); err != nil {
				return fmt.Errorf("invalid version id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowVersion(cmd.Context(), workflowID, versionID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&versionID, "version", "", "version ID (required)")
	mustMarkFlagRequired(cmd, "version")
	return cmd
}

func newWorkflowsVersionImpactCommand(state *appState) *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "version-impact <workflow-id-or-slug>",
		Short: "Get the impact analysis for a specific workflow version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(versionID); err != nil {
				return fmt.Errorf("invalid version id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowVersionImpact(cmd.Context(), workflowID, versionID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&versionID, "version", "", "version ID (required)")
	mustMarkFlagRequired(cmd, "version")
	return cmd
}

func newWorkflowsVersionStepsCommand(state *appState) *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "version-steps <workflow-id-or-slug>",
		Short: "Get the steps for a specific workflow version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(versionID); err != nil {
				return fmt.Errorf("invalid version id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowVersionSteps(cmd.Context(), workflowID, versionID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&versionID, "version", "", "version ID (required)")
	mustMarkFlagRequired(cmd, "version")
	return cmd
}

func newWorkflowsCanaryGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "canary-get <workflow-id-or-slug>",
		Short: "Get the canary configuration for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowCanary(cmd.Context(), workflowID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowsCanarySetCommand(state *appState) *cobra.Command {
	var trafficPct float64
	cmd := &cobra.Command{
		Use:   "canary-set <workflow-id-or-slug>",
		Short: "Set the canary traffic percentage for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.SetWorkflowCanary(cmd.Context(), workflowID, trafficPct)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintf(os.Stderr, "%s\n", styles.Success("Canary traffic set to "+fmt.Sprintf("%.2f", trafficPct)+"% for "+styles.Bold.Render(workflowID)))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().Float64Var(&trafficPct, "traffic-pct", 0, "canary traffic percentage (required)")
	mustMarkFlagRequired(cmd, "traffic-pct")
	return cmd
}

func newWorkflowsCanaryRollbackCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "canary-rollback <workflow-id-or-slug>",
		Short: "Roll back the canary for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			out, err := cli.RollbackWorkflowCanary(cmd.Context(), workflowID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Canary rolled back for "+styles.Bold.Render(workflowID)))
			}
			return printData(state, out)
		},
	}
}

// registerWorkflowRunsCoverageCommands adds coverage subcommands to the
// workflow-runs parent command. It does not duplicate any subcommands already
// registered by newWorkflowRunsCommand (list, get, steps, approve, cancel,
// force-complete, pause, resume, retry, skip, watch).
func registerWorkflowRunsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newWorkflowRunsCompareCommand(state))
	parent.AddCommand(newWorkflowRunsCompensationPlanCommand(state))
	parent.AddCommand(newWorkflowRunsDebugCommand(state))
	parent.AddCommand(newWorkflowRunsExplainCommand(state))
	parent.AddCommand(newWorkflowRunsGraphCommand(state))
	parent.AddCommand(newWorkflowRunsLabelsCommand(state))
	parent.AddCommand(newWorkflowRunsTimelineCommand(state))
	parent.AddCommand(newWorkflowRunsCompensateCommand(state))
	parent.AddCommand(newWorkflowRunsReplaySubtreeCommand(state))
	parent.AddCommand(newWorkflowRunsBulkCancelCommand(state))
	parent.AddCommand(newWorkflowRunsBulkReplayCommand(state))
}

func newWorkflowRunsCompareCommand(state *appState) *cobra.Command {
	var otherID string
	cmd := &cobra.Command{
		Use:   "compare <workflow-run-id>",
		Short: "Compare a workflow run with another run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(otherID); err != nil {
				return fmt.Errorf("invalid --other run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CompareWorkflowRuns(cmd.Context(), args[0], otherID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&otherID, "other", "", "ID of the other workflow run to compare against (required)")
	mustMarkFlagRequired(cmd, "other")
	return cmd
}

func newWorkflowRunsCompensationPlanCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "compensation-plan <workflow-run-id>",
		Short: "Get the compensation plan for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunCompensationPlan(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsDebugCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "debug <workflow-run-id>",
		Short: "Get debug information for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunDebug(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsExplainCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <workflow-run-id>",
		Short: "Get an explanation for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunExplain(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsGraphCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "graph <workflow-run-id>",
		Short: "Get the execution graph for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunGraph(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsLabelsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "labels <workflow-run-id>",
		Short: "Get the labels for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunLabels(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsTimelineCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <workflow-run-id>",
		Short: "Get the timeline for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetWorkflowRunTimeline(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsCompensateCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "compensate <workflow-run-id>",
		Short: "Trigger compensation for a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CompensateWorkflowRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Compensation triggered for workflow run "+styles.Bold.Render(args[0])))
			}
			return printData(state, out)
		},
	}
}

func newWorkflowRunsReplaySubtreeCommand(state *appState) *cobra.Command {
	var stepRef string
	cmd := &cobra.Command{
		Use:   "replay-subtree <workflow-run-id>",
		Short: "Replay a step subtree within a workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(stepRef); err != nil {
				return fmt.Errorf("invalid step ref: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ReplayWorkflowStepSubtree(cmd.Context(), args[0], stepRef)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Replayed subtree for step "+styles.Bold.Render(stepRef)+" on run "+styles.Bold.Render(args[0])))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&stepRef, "step", "", "step ref to replay from (required)")
	mustMarkFlagRequired(cmd, "step")
	return cmd
}

func newWorkflowRunsBulkCancelCommand(state *appState) *cobra.Command {
	var ids []string
	cmd := &cobra.Command{
		Use:   "bulk-cancel",
		Short: "Cancel multiple workflow runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(ids) == 0 {
				return fmt.Errorf("at least one --id is required")
			}
			for _, id := range ids {
				if err := validate.SlugOrID(id); err != nil {
					return fmt.Errorf("invalid workflow run id %q: %w", id, err)
				}
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkCancelWorkflowRuns(cmd.Context(), ids)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Cancelled %d workflow run(s)", len(ids))))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&ids, "id", nil, "workflow run ID to cancel (repeatable)")
	mustMarkFlagRequired(cmd, "id")
	return cmd
}

func newWorkflowRunsBulkReplayCommand(state *appState) *cobra.Command {
	var ids []string
	cmd := &cobra.Command{
		Use:   "bulk-replay",
		Short: "Replay multiple workflow runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(ids) == 0 {
				return fmt.Errorf("at least one --id is required")
			}
			for _, id := range ids {
				if err := validate.SlugOrID(id); err != nil {
					return fmt.Errorf("invalid workflow run id %q: %w", id, err)
				}
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkReplayWorkflowRuns(cmd.Context(), ids)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Replayed %d workflow run(s)", len(ids))))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&ids, "id", nil, "workflow run ID to replay (repeatable)")
	mustMarkFlagRequired(cmd, "id")
	return cmd
}
