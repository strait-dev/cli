package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newCodeDeploymentsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployments",
		Short: "Manage code-first job deployments",
	}

	cmd.AddCommand(newCodeDeploymentsListCommand(state))

	getCmd, getJobSlug := newCodeDeploymentGetCommand(state)
	getCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *getJobSlug })
	cmd.AddCommand(getCmd)

	logsCmd, logsJobSlug := newCodeDeploymentLogsCommand(state)
	logsCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *logsJobSlug })
	cmd.AddCommand(logsCmd)

	rollbackCmd, rollbackJobSlug := newCodeDeploymentRollbackCommand(state)
	rollbackCmd.ValidArgsFunction = completeDeploymentIDs(state, func() string { return *rollbackJobSlug })
	cmd.AddCommand(rollbackCmd)

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
				return cli.StreamDeploymentLogs(cmd.Context(), job.ID, deploymentID, func(chunk string) error {
					fmt.Fprint(os.Stdout, chunk)
					return nil
				})
			}

			fmt.Print(d.BuildLogs)
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
