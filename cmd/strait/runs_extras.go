package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newRunsRescheduleCommand(state *appState) *cobra.Command {
	var at string
	cmd := &cobra.Command{
		Use:   "reschedule <run-id>",
		Short: "Reschedule a run for a future execution time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			if strings.TrimSpace(at) == "" {
				return fmt.Errorf("--at is required (RFC3339)")
			}
			ts, err := time.Parse(time.RFC3339, at)
			if err != nil {
				return fmt.Errorf("--at must be RFC3339: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.RescheduleRun(cmd.Context(), args[0], ts)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Rescheduled run "+styles.Bold.Render(args[0])+" for "+styles.TimestampFull(ts)))
				return nil
			}
			return printData(state, run)
		},
	}
	cmd.Flags().StringVar(&at, "at", "", "RFC3339 timestamp to reschedule the run for (required)")
	_ = cmd.MarkFlagRequired("at")
	return cmd
}

func newRunsDLQCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "dlq",
		Short: "List runs in the dead letter queue",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			items, err := cli.ListDLQ(cmd.Context(), pid)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("DLQ", len(items)))
				for _, d := range items {
					fmt.Fprintf(os.Stderr, "  %s  job=%s  reason=%s  %s\n",
						styles.Bold.Render(styles.SafeText(d.ID)),
						styles.MutedStyle.Render(styles.SafeText(d.JobID)),
						styles.SafeText(d.Reason),
						styles.RelativeTime(d.FailedAt),
					)
				}
				return nil
			}
			return printData(state, items)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	return cmd
}

func newRunsDLQReplayCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "dlq-replay <dlq-id>",
		Short: "Replay a run from the dead letter queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid dlq id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.ReplayDLQ(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Replayed DLQ entry "+styles.Bold.Render(styles.SafeText(args[0]))+" as run "+styles.Bold.Render(styles.SafeText(run.ID))))
				return nil
			}
			return printData(state, run)
		},
	}
}

func newRunsOutputsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "outputs <run-id>",
		Short: "List outputs produced by a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			outputs, err := cli.ListRunOutputs(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, outputs)
		},
	}
}

func newRunsToolCallsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "tool-calls <run-id>",
		Short: "List tool calls invoked during a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			calls, err := cli.ListRunToolCalls(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, calls)
		},
	}
}

func newRunsUsageCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "usage <run-id>",
		Short: "Show resource usage for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			usage, err := cli.GetRunUsage(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Run", usage.RunID),
					styles.DetailLine("Duration ms", fmt.Sprintf("%d", usage.DurationMS)),
					styles.DetailLine("CPU seconds", fmt.Sprintf("%.2f", usage.CPUSeconds)),
					styles.DetailLine("Memory MB-hours", fmt.Sprintf("%.4f", usage.MemoryMBHours)),
					styles.DetailLine("Tokens in", fmt.Sprintf("%d", usage.TokensInput)),
					styles.DetailLine("Tokens out", fmt.Sprintf("%d", usage.TokensOutput)),
					styles.DetailLine("Cost (USD)", fmt.Sprintf("%.4f", usage.CostUSD)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Run Usage", lines))
				return nil
			}
			return printData(state, usage)
		},
	}
}

func newRunsCheckpointsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "checkpoints <run-id>",
		Short: "List checkpoints recorded during a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			checkpoints, err := cli.ListRunCheckpoints(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, checkpoints)
		},
	}
}
