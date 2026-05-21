package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/types"

	"github.com/spf13/cobra"
)

var (
	runsTimeNow = time.Now
	runsAfter   = time.After
)

func newRunsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Manage runs",
	}

	cmd.AddCommand(newRunsListCommand(state))
	cmd.AddCommand(newRunsGetCommand(state))
	cmd.AddCommand(newRunsLogsCommand(state))
	cmd.AddCommand(newRunsCancelCommand(state))
	cmd.AddCommand(newRunsReplayCommand(state))
	cmd.AddCommand(newRunsRescheduleCommand(state))
	cmd.AddCommand(newRunsDLQCommand(state))
	cmd.AddCommand(newRunsDLQReplayCommand(state))
	cmd.AddCommand(newRunsOutputsCommand(state))
	cmd.AddCommand(newRunsToolCallsCommand(state))
	cmd.AddCommand(newRunsCheckpointsCommand(state))
	cmd.AddCommand(newRunsWatchCommand(state))

	return cmd
}

func newRunsListCommand(state *appState) *cobra.Command {
	var projectID string
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			runs, err := cli.ListRuns(cmd.Context(), projectID, status, limit, nil)
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(runs))
			for _, run := range runs {
				rows = append(rows, map[string]any{
					"id":           run.ID,
					"job_id":       run.JobID,
					"status":       styles.Status(string(run.Status)),
					"attempt":      run.Attempt,
					"triggered_by": run.TriggeredBy,
					"created_at":   run.CreatedAt,
				})
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Runs", len(runs)))
				for _, run := range runs {
					fmt.Fprintf(os.Stderr, "  %s  %s  job=%s  attempt=%d  by=%s  %s\n",
						styles.StatusBadge(string(run.Status)),
						run.ID,
						styles.MutedStyle.Render(run.JobID),
						run.Attempt,
						run.TriggeredBy,
						styles.RelativeTime(run.CreatedAt),
					)
				}
				return nil
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().IntVar(&limit, "limit", 50, "max runs to return")
	_ = cmd.RegisterFlagCompletionFunc("status", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"delayed", "queued", "dequeued", "executing", "waiting", "completed", "failed", "timed_out", "crashed", "system_failed", "canceled", "expired"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newRunsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <run-id>",
		Short: "Get run by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.GetRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Status", styles.StatusBadge(string(run.Status))),
					styles.DetailLine("ID", run.ID),
					styles.DetailLine("Job", run.JobID),
					styles.DetailLine("Attempt", fmt.Sprintf("%d", run.Attempt)),
					styles.DetailLine("Triggered", run.TriggeredBy),
					styles.DetailLine("Created", styles.TimestampFull(run.CreatedAt)),
				}
				if run.StartedAt != nil {
					lines = append(lines, styles.DetailLine("Started", styles.TimestampFull(*run.StartedAt)))
				}
				if run.FinishedAt != nil {
					lines = append(lines, styles.DetailLine("Finished", styles.TimestampFull(*run.FinishedAt)))
				}
				if run.Error != "" {
					lines = append(lines, styles.DetailLine("Error", styles.Red.Render(run.Error)))
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Run", lines))
				return nil
			}
			return printData(state, run)
		},
	}

	return cmd
}

func newRunsCancelCommand(state *appState) *cobra.Command {
	var all bool
	var projectID string
	var status string
	var limit int
	var yes bool

	cmd := &cobra.Command{
		Use:   "cancel <run-id> [run-id...]",
		Short: "Cancel one or more runs",
		Args: func(_ *cobra.Command, args []string) error {
			if all || len(args) > 0 {
				return nil
			}
			return fmt.Errorf("provide run IDs or use --all")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			targetIDs := make([]string, 0)
			if all {
				projectID, err = requireProjectID(state, projectID)
				if err != nil {
					return err
				}
				runs, listErr := cli.ListRuns(cmd.Context(), projectID, status, limit, nil)
				if listErr != nil {
					return listErr
				}
				for _, run := range runs {
					targetIDs = append(targetIDs, run.ID)
				}
			} else {
				targetIDs = append(targetIDs, args...)
			}

			if len(targetIDs) == 0 {
				return fmt.Errorf("no runs matched cancellation criteria")
			}
			if len(targetIDs) > 1 {
				if err := requireConfirmation(state, fmt.Sprintf("Cancel %d runs?", len(targetIDs)), yes); err != nil {
					return err
				}
			}

			results := make([]map[string]any, 0, len(targetIDs))
			if len(targetIDs) == 1 {
				id := targetIDs[0]
				run, cancelErr := cli.CancelRun(cmd.Context(), id)
				if cancelErr != nil {
					results = append(results, map[string]any{"id": id, "canceled": false, "error": cancelErr.Error()})
					if isTTYRich(state) {
						fmt.Fprintln(os.Stderr, styles.Err("Failed to cancel "+id+": "+cancelErr.Error()))
					}
				} else {
					results = append(results, map[string]any{"id": id, "canceled": true, "status": run.Status})
					if isTTYRich(state) {
						fmt.Fprintln(os.Stderr, styles.Success("Canceled run "+styles.Bold.Render(id)))
					}
				}
			} else {
				resp, bulkErr := cli.BulkCancelRuns(cmd.Context(), targetIDs)
				if bulkErr != nil {
					return fmt.Errorf("bulk cancel runs: %w", bulkErr)
				}
				for _, r := range resp.Results {
					if r.Canceled {
						results = append(results, map[string]any{"id": r.ID, "canceled": true, "status": r.Status})
						if isTTYRich(state) {
							fmt.Fprintln(os.Stderr, styles.Success("Canceled run "+styles.Bold.Render(r.ID)))
						}
					} else {
						results = append(results, map[string]any{"id": r.ID, "canceled": false, "error": r.Error})
						if isTTYRich(state) {
							fmt.Fprintln(os.Stderr, styles.Err("Failed to cancel "+r.ID+": "+r.Error))
						}
					}
				}
			}

			if isTTYRich(state) {
				return nil
			}
			return printData(state, results)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "cancel all runs matching filters")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID for --all mode")
	cmd.Flags().StringVar(&status, "status", "", "status filter for --all mode")
	cmd.Flags().IntVar(&limit, "limit", 100, "max runs to consider for --all mode")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm bulk cancellation")

	return cmd
}

func newRunsLogsCommand(state *appState) *cobra.Command {
	var follow bool
	var level string
	var eventType string

	cmd := &cobra.Command{
		Use:   "logs <run-id>",
		Short: "Show run events/logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if !follow {
				rows, err := listRunEventRows(ctx, cli, args[0], level, eventType, "", time.Time{})
				if err != nil {
					return err
				}
				return printLogRows(state, rows, false, "", 0)
			}

			if err := ensureRunStreamable(ctx, cli, args[0]); err != nil {
				return err
			}

			rows, err := listRunEventRows(ctx, cli, args[0], level, eventType, "", time.Time{})
			if err != nil {
				return err
			}
			if err := printLogRows(state, rows, false, "", 0); err != nil {
				return err
			}

			return streamRunLogs(ctx, cli, state, args[0], level, eventType, "", time.Time{}, "")
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream logs over SSE")
	cmd.Flags().StringVar(&level, "level", "", "event level filter")
	cmd.Flags().StringVar(&eventType, "type", "", "event type filter")
	_ = cmd.RegisterFlagCompletionFunc("level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"log", "state_change", "error", "progress"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newRunsWatchCommand(state *appState) *cobra.Command {
	var interval time.Duration
	var timeout time.Duration
	var until string

	cmd := &cobra.Command{
		Use:   "watch <run-id>",
		Short: "Watch a run until it reaches a terminal state",
		Long: `Polls a run until it reaches a terminal state, then exits.

By default exits 0 only when the run completes successfully.
Use --until to accept specific terminal statuses as success (e.g. --until completed,failed).`,
		Args: cobra.ExactArgs(1),
		Example: `  strait runs watch run-abc123
  strait runs watch run-abc123 --until completed,failed
  strait runs watch run-abc123 --timeout 10m --interval 5s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			ttyMode := isTTYRich(state)

			// Parse --until into a set of accepted terminal statuses.
			acceptedStatuses := parseUntilStatuses(until)

			deadline := runsTimeNow().Add(timeout)
			for {
				run, err := cli.GetRun(ctx, args[0])
				if err != nil {
					return err
				}

				if ttyMode {
					fmt.Fprintf(os.Stderr, "\r%s %s  attempt=%d",
						styles.StatusBadge(string(run.Status)), run.ID, run.Attempt)
				} else if err := printData(state, map[string]any{
					"id":      run.ID,
					"status":  run.Status,
					"attempt": run.Attempt,
				}); err != nil {
					return err
				}

				if run.Status.IsTerminal() {
					if ttyMode {
						fmt.Fprintln(os.Stderr)
					}
					// If --until was specified, succeed when run status is in accepted set.
					if len(acceptedStatuses) > 0 {
						if acceptedStatuses[string(run.Status)] {
							if ttyMode {
								fmt.Fprintln(os.Stderr, styles.Success("Run reached status "+string(run.Status)))
							}
							return nil
						}
						if ttyMode {
							fmt.Fprintln(os.Stderr, styles.Err("Run reached status "+string(run.Status)+" (not in --until set)"))
						}
						return fmt.Errorf("run reached terminal status %q", run.Status)
					}
					// Default: only completed is success.
					if run.Status == types.StatusCompleted {
						if ttyMode {
							fmt.Fprintln(os.Stderr, styles.Success("Run completed"))
						}
						return nil
					}
					if ttyMode {
						fmt.Fprintln(os.Stderr, styles.Err("Run reached terminal status "+string(run.Status)))
					}
					return fmt.Errorf("run reached terminal status %q", run.Status)
				}

				if timeout > 0 && runsTimeNow().After(deadline) {
					if ttyMode {
						fmt.Fprintln(os.Stderr)
					}
					return fmt.Errorf("watch timeout reached")
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-runsAfter(interval):
				}
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "poll interval")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "max watch duration (0 disables timeout)")
	cmd.Flags().StringVar(&until, "until", "", "comma-separated list of terminal statuses to treat as success (e.g. completed,failed)")

	return cmd
}

// parseUntilStatuses parses a comma-separated list of run statuses into a lookup set.
// Returns nil if the input is empty.
func parseUntilStatuses(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	m := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

// watchRunUntilDone polls a run until it reaches a terminal state. It is used by
// trigger --wait and replay --wait to avoid synthesizing a cobra command context.
func watchRunUntilDone(ctx context.Context, state *appState, runID string, interval, timeout time.Duration) error {
	cli, err := newAPIClient(state)
	if err != nil {
		return err
	}

	ttyMode := isTTYRich(state)
	deadline := runsTimeNow().Add(timeout)
	for {
		run, err := cli.GetRun(ctx, runID)
		if err != nil {
			return err
		}

		if ttyMode {
			fmt.Fprintf(os.Stderr, "\r%s %s  attempt=%d",
				styles.StatusBadge(string(run.Status)), run.ID, run.Attempt)
		} else if err := printData(state, map[string]any{
			"id":      run.ID,
			"status":  run.Status,
			"attempt": run.Attempt,
		}); err != nil {
			return err
		}

		if run.Status.IsTerminal() {
			if ttyMode {
				fmt.Fprintln(os.Stderr)
				if run.Status == types.StatusCompleted {
					fmt.Fprintln(os.Stderr, styles.Success("Run completed"))
				} else {
					fmt.Fprintln(os.Stderr, styles.Err("Run reached terminal status "+string(run.Status)))
				}
			}
			if run.Status == types.StatusCompleted {
				return nil
			}
			return fmt.Errorf("run reached terminal status %q", run.Status)
		}

		if timeout > 0 && runsTimeNow().After(deadline) {
			return fmt.Errorf("watch timeout reached")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-runsAfter(interval):
		}
	}
}

func newRunsReplayCommand(state *appState) *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:   "replay <run-id>",
		Short: "Replay a run, preserving lineage to the original",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			replayed, err := cli.ReplayRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info("Replayed as run "+styles.Bold.Render(replayed.ID)))
			} else if err := printData(state, replayed); err != nil {
				return err
			}

			if !wait {
				return nil
			}

			return watchRunUntilDone(cmd.Context(), state, replayed.ID, 2*time.Second, 5*time.Minute)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "wait for replayed run to reach terminal state")

	return cmd
}
