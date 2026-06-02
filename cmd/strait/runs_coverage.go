package main

import (
	"fmt"

	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

// registerRunsCoverageCommands attaches extended run subcommands to the parent
// runs command. It must not duplicate subcommands already registered in
// runs.go or runs_extras.go (list, get, logs, outputs, checkpoints, tool-calls,
// replay, cancel, reschedule, dlq, dlq-replay, watch).
func registerRunsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newRunsChildrenCommand(state))
	parent.AddCommand(newRunsLineageCommand(state))
	parent.AddCommand(newRunsDependencyStatusCommand(state))
	parent.AddCommand(newRunsStateCommand(state))
	parent.AddCommand(newRunsResourcesCommand(state))
	parent.AddCommand(newRunsDebugBundleCommand(state))
	parent.AddCommand(newRunsUsageCommand(state))
	parent.AddCommand(newRunsRestartCommand(state))
	parent.AddCommand(newRunsPauseCommand(state))
	parent.AddCommand(newRunsResumeCommand(state))
	parent.AddCommand(newRunsDebugCommand(state))
	parent.AddCommand(newRunsResetIdempotencyKeyCommand(state))
	parent.AddCommand(newRunsBulkCancelCommand(state))
	parent.AddCommand(newRunsBulkCancelAllCommand(state))
	parent.AddCommand(newRunsBulkReplayCommand(state))
	parent.AddCommand(newRunsBulkDLQReplayCommand(state))
}

func newRunsChildrenCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "children <run-id>",
		Short: "List child runs spawned by a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRunChildren(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsLineageCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "lineage <run-id>",
		Short: "Get the lineage graph for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetRunLineage(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsDependencyStatusCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "dependency-status <run-id>",
		Short: "Get the dependency status for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetRunDependencyStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsStateCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "state <run-id>",
		Short: "List state entries for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRunState(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsResourcesCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "resources <run-id>",
		Short: "List resources associated with a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRunResources(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsDebugBundleCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "debug-bundle <run-id>",
		Short: "Get the debug bundle for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetRunDebugBundle(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsUsageCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "usage <run-id>",
		Short: "List usage records for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRunUsage(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsRestartCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "restart <run-id>",
		Short: "Restart a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.RestartRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsPauseCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <run-id>",
		Short: "Pause a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.PauseRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsResumeCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <run-id>",
		Short: "Resume a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ResumeRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRunsDebugCommand(state *appState) *cobra.Command {
	var enable bool
	cmd := &cobra.Command{
		Use:   "debug <run-id>",
		Short: "Enable or disable debug mode on a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.SetRunDebug(cmd.Context(), args[0], enable)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().BoolVar(&enable, "enable", true, "enable (true) or disable (false) debug mode")
	return cmd
}

func newRunsResetIdempotencyKeyCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "reset-idempotency-key <run-id>",
		Short: "Delete the idempotency key for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.ResetRunIdempotencyKey(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printData(state, map[string]any{"reset": true, "id": args[0]})
		},
	}
}

func newRunsBulkCancelCommand(state *appState) *cobra.Command {
	var runIDs []string
	cmd := &cobra.Command{
		Use:   "bulk-cancel",
		Short: "Cancel multiple runs by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(runIDs) == 0 {
				return fmt.Errorf("--id is required (specify one or more run IDs)")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkCancelRunsByIDs(cmd.Context(), runIDs)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&runIDs, "id", nil, "run ID to cancel (repeatable)")
	return cmd
}

func newRunsBulkCancelAllCommand(state *appState) *cobra.Command {
	var jobID, status, batchID, triggeredBy string
	cmd := &cobra.Command{
		Use:   "bulk-cancel-all",
		Short: "Cancel all runs matching optional filters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkCancelAllRuns(cmd.Context(), jobID, status, batchID, triggeredBy)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&jobID, "job-id", "", "filter by job ID")
	cmd.Flags().StringVar(&status, "status", "", "filter by run status")
	cmd.Flags().StringVar(&batchID, "batch-id", "", "filter by batch ID")
	cmd.Flags().StringVar(&triggeredBy, "triggered-by", "", "filter by triggered_by value")
	return cmd
}

func newRunsBulkReplayCommand(state *appState) *cobra.Command {
	var runIDs []string
	cmd := &cobra.Command{
		Use:   "bulk-replay",
		Short: "Replay multiple runs by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(runIDs) == 0 {
				return fmt.Errorf("--id is required (specify one or more run IDs)")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkReplayRuns(cmd.Context(), runIDs)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&runIDs, "id", nil, "run ID to replay (repeatable)")
	return cmd
}

func newRunsBulkDLQReplayCommand(state *appState) *cobra.Command {
	var runIDs []string
	var projectID string
	var limit int
	cmd := &cobra.Command{
		Use:   "bulk-dlq-replay",
		Short: "Replay dead-letter-queue runs in bulk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(runIDs) == 0 {
				return fmt.Errorf("--id is required (specify one or more run IDs)")
			}
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkDLQReplayRuns(cmd.Context(), runIDs, pid, limit)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&runIDs, "id", nil, "run ID to replay from DLQ (repeatable)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().IntVar(&limit, "limit", 0, "max runs to replay")
	return cmd
}
