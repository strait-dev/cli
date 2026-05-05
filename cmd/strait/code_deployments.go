package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/codedeploy"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

var watchCodeDeploymentUntilTerminal = codedeploy.WatchUntilTerminal

func newCodeDeploymentsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployments",
		Short: "Manage code-first job deployments",
	}

	cmd.AddCommand(newCodeDeploymentsListCommand(state))
	cmd.AddCommand(newCodeDeploymentsCreateFromSourceCommand(state))

	getCmd, getJobSlug := newCodeDeploymentGetCommand(state)
	getCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *getJobSlug })
	cmd.AddCommand(getCmd)

	logsCmd, logsJobSlug := newCodeDeploymentLogsCommand(state)
	logsCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *logsJobSlug })
	cmd.AddCommand(logsCmd)

	rollbackCmd, rollbackJobSlug := newCodeDeploymentRollbackCommand(state)
	rollbackCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *rollbackJobSlug })
	cmd.AddCommand(rollbackCmd)

	watchCmd, watchJobSlug := newCodeDeploymentWatchCommand(state)
	watchCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *watchJobSlug })
	cmd.AddCommand(watchCmd)

	return cmd
}

func newCodeDeploymentsListCommand(state *appState) *cobra.Command {
	var jobSlug, projectID string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List code deployments for a job",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			job, err := cli.GetJobBySlug(cmd.Context(), resolvedProject, jobSlug)
			if err != nil {
				return fmt.Errorf("look up job: %w", err)
			}

			deps, err := cli.ListCodeDeployments(cmd.Context(), job.ID, limit)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Deployments", len(deps)))
				for _, d := range deps {
					fmt.Fprintf(os.Stderr, "  %s  v%-3d  %s  %s  %s\n",
						styles.Bold.Render(d.ID),
						d.Version,
						d.Runtime,
						styles.StatusBadge(d.Status),
						styles.RelativeTime(d.CreatedAt),
					)
				}
				return nil
			}

			rows := make([]map[string]any, 0, len(deps))
			for _, d := range deps {
				rows = append(rows, deploymentToRow(d))
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug (required)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "max deployments to show")

	return cmd
}

func newCodeDeploymentGetCommand(state *appState) (*cobra.Command, *string) {
	var jobSlug, projectID string

	cmd := &cobra.Command{
		Use:   "get <deployment-id>",
		Short: "Get a code deployment by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			if err := validate.ResourceID(args[0]); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}

			job, err := cli.GetJobBySlug(cmd.Context(), resolvedProject, jobSlug)
			if err != nil {
				return fmt.Errorf("look up job: %w", err)
			}

			d, err := cli.GetCodeDeployment(cmd.Context(), job.ID, args[0])
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Bold.Render(d.ID))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Job", d.JobID))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Version", fmt.Sprintf("%d", d.Version)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Status", styles.StatusBadge(d.Status)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Runtime", d.Runtime))
				if d.BuiltImageURI != "" {
					fmt.Fprintln(os.Stderr, styles.KeyValue("Image", d.BuiltImageURI))
				}
				if d.ErrorMessage != "" {
					fmt.Fprintln(os.Stderr, styles.KeyValue("Error", d.ErrorMessage))
				}
				fmt.Fprintln(os.Stderr, styles.KeyValue("Created", styles.RelativeTime(d.CreatedAt)))
				return nil
			}
			return printData(state, deploymentToRow(*d))
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug (required)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")

	return cmd, &jobSlug
}

func newCodeDeploymentLogsCommand(state *appState) (*cobra.Command, *string) {
	var jobSlug, projectID string
	var stream bool

	cmd := &cobra.Command{
		Use:   "logs <deployment-id>",
		Short: "Get or stream build logs for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			if err := validate.ResourceID(args[0]); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}

			job, err := cli.GetJobBySlug(cmd.Context(), resolvedProject, jobSlug)
			if err != nil {
				return fmt.Errorf("look up job: %w", err)
			}

			deploymentID := args[0]

			// Fetch deployment to check current status.
			d, err := cli.GetCodeDeployment(cmd.Context(), job.ID, deploymentID)
			if err != nil {
				return err
			}

			// Auto-stream when the build is still running and caller did not
			// explicitly request non-streaming output.
			wantStream := stream || d.Status == "building"
			if wantStream {
				w := state.out()
				return cli.StreamDeploymentLogs(cmd.Context(), job.ID, deploymentID, func(chunk string) error {
					fmt.Fprint(w, chunk)
					return nil
				})
			}

			fmt.Fprint(state.out(), d.BuildLogs)
			return nil
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug (required)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().BoolVar(&stream, "stream", false, "stream logs in real time (for in-progress builds)")

	return cmd, &jobSlug
}

func newCodeDeploymentRollbackCommand(state *appState) (*cobra.Command, *string) {
	var jobSlug, projectID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "rollback <deployment-id>",
		Short: "Roll back to a previous ready deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			if err := validate.ResourceID(args[0]); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			deploymentID := args[0]

			if confirmErr := requireConfirmation(state,
				fmt.Sprintf("Roll back job %s to deployment %s?", jobSlug, deploymentID),
				yes,
			); confirmErr != nil {
				return confirmErr
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			job, err := cli.GetJobBySlug(cmd.Context(), resolvedProject, jobSlug)
			if err != nil {
				return fmt.Errorf("look up job: %w", err)
			}

			d, err := cli.RollbackCodeDeployment(cmd.Context(), job.ID, deploymentID, resolvedProject)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf(
					"Rolled back job %s to deployment %s (v%d)",
					jobSlug, styles.Bold.Render(d.ID), d.Version,
				)))
				return nil
			}
			return printData(state, map[string]any{
				"job":           jobSlug,
				"deployment_id": d.ID,
				"version":       d.Version,
				"status":        d.Status,
			})
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug (required)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")

	return cmd, &jobSlug
}

func newCodeDeploymentWatchCommand(state *appState) (*cobra.Command, *string) {
	var jobSlug, projectID string
	var watchTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "watch <deployment-id>",
		Short: "Follow a deployment to completion and exit with its result",
		Long: `Streams build logs in real time (if the build is still in progress).
Falls back to polling when the log stream is unavailable.

Exits 0 when the deployment reaches 'ready', exits 1 on 'failed' or 'timed_out'.
Safe to use in CI pipelines and agent workflows.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			if err := validate.ResourceID(args[0]); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			job, err := cli.GetJobBySlug(cmd.Context(), resolvedProject, jobSlug)
			if err != nil {
				return fmt.Errorf("look up job: %w", err)
			}

			deploymentID := args[0]

			// Fetch current status — if already terminal, exit immediately.
			d, err := cli.GetCodeDeployment(cmd.Context(), job.ID, deploymentID)
			if err != nil {
				return err
			}

			switch d.Status {
			case "ready":
				if isTTYRich(state) {
					fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Deployment %s is ready", styles.Bold.Render(d.ID))))
					if d.BuiltImageURI != "" {
						fmt.Fprintln(os.Stderr, styles.KeyValue("Image", d.BuiltImageURI))
					}
				} else {
					_ = printData(state, deploymentToRow(*d))
				}
				return nil
			case "failed", "timed_out":
				msg := d.ErrorMessage
				if msg == "" {
					msg = d.Status
				}
				return fmt.Errorf("deployment %s %s: %s", d.ID, d.Status, msg)
			}

			// Build is in progress — apply watch timeout to context.
			ctx := cmd.Context()
			if watchTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, watchTimeout)
				defer cancel()
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info(fmt.Sprintf("Watching deployment %s (status: %s)...", styles.Bold.Render(d.ID), d.Status)))
			}

			// Try streaming logs first.
			streamCompleted := false
			w := state.out()
			streamErr := cli.StreamDeploymentLogs(ctx, job.ID, deploymentID, func(chunk string) error {
				fmt.Fprint(w, chunk)
				return nil
			})
			if streamErr == nil {
				streamCompleted = true
			} else if ctx.Err() != nil {
				return fmt.Errorf("watch timed out after %s", watchTimeout)
			} else if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.MutedStyle.Render(fmt.Sprintf("Log stream ended (%v), polling for completion...", streamErr)))
			}

			// Fetch final status.
			var final *client.CodeDeployment
			if streamCompleted {
				final, err = cli.GetCodeDeployment(ctx, job.ID, deploymentID)
				if err != nil {
					return fmt.Errorf("fetch final status: %w", err)
				}
			} else {
				// Poll until terminal, emitting NDJSON status ticks when not in TTY mode.
				final, err = watchCodeDeploymentUntilTerminal(ctx, cli, job.ID, deploymentID, func(status string, elapsed time.Duration) {
					if isTTYRich(state) && !state.opts.quiet {
						fmt.Fprintf(os.Stderr, "  status: %s (elapsed: %s)\n",
							styles.StatusBadge(status),
							elapsed.Round(time.Second),
						)
					} else if !state.opts.quiet {
						// NDJSON tick — machine-readable for CI pipelines.
						_ = printData(state, map[string]any{
							"status":      status,
							"elapsed_sec": int(elapsed.Seconds()),
						})
					}
				})
				if err != nil {
					return err
				}
			}

			switch final.Status {
			case "ready":
				if isTTYRich(state) {
					fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Deployment %s ready", styles.Bold.Render(final.ID))))
					if final.BuiltImageURI != "" {
						fmt.Fprintln(os.Stderr, styles.KeyValue("Image", final.BuiltImageURI))
					}
				} else {
					_ = printData(state, deploymentToRow(*final))
				}
				return nil
			default:
				msg := final.ErrorMessage
				if msg == "" {
					msg = final.Status
				}
				return fmt.Errorf("deployment %s %s: %s", final.ID, final.Status, msg)
			}
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug (required)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().DurationVar(&watchTimeout, "timeout", 30*time.Minute, "maximum time to wait for the deployment to complete")

	return cmd, &jobSlug
}

func deploymentToRow(d client.CodeDeployment) map[string]any {
	row := map[string]any{
		"id":                d.ID,
		"job_id":            d.JobID,
		"version":           d.Version,
		"status":            d.Status,
		"runtime":           d.Runtime,
		"source_size_bytes": d.SourceSizeBytes,
		"created_at":        d.CreatedAt,
	}
	if d.BuiltImageURI != "" {
		row["built_image_uri"] = d.BuiltImageURI
	}
	if d.ErrorMessage != "" {
		row["error_message"] = d.ErrorMessage
	}
	if d.FinishedAt != nil {
		row["finished_at"] = d.FinishedAt
	}
	return row
}
