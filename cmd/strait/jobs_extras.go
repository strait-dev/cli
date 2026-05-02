package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newJobsCloneCommand(state *appState) *cobra.Command {
	var name string
	var slug string

	cmd := &cobra.Command{
		Use:   "clone <job-id-or-slug>",
		Short: "Clone an existing job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			job, err := cli.CloneJob(cmd.Context(), id, client.CloneJobRequest{Name: name, Slug: slug})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Cloned to "+styles.Bold.Render(job.Slug)))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name for the cloned job")
	cmd.Flags().StringVar(&slug, "slug", "", "slug for the cloned job")

	return cmd
}

func newJobsHealthCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health <job-id-or-slug>",
		Short: "Show health summary for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			health, err := cli.GetJobHealth(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Job", health.JobID),
					styles.DetailLine("Status", health.Status),
					styles.DetailLine("Last run", health.LastRunStatus),
					styles.DetailLine("Success rate", fmt.Sprintf("%.2f%%", health.SuccessRate*100)),
					styles.DetailLine("p95", fmt.Sprintf("%dms", health.P95DurationMS)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job Health", lines))
				return nil
			}
			return printData(state, health)
		},
	}
	return cmd
}

func newJobsDependenciesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependencies <job-id-or-slug>",
		Short: "List dependency edges for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			deps, err := cli.ListJobDependencies(cmd.Context(), id)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(deps))
			for _, d := range deps {
				rows = append(rows, map[string]any{
					"id":         d.ID,
					"depends_on": d.DependsOn,
					"type":       d.Type,
					"created_at": d.CreatedAt,
				})
			}
			return printData(state, rows)
		},
	}
	return cmd
}

func newJobsAddDependencyCommand(state *appState) *cobra.Command {
	var dependsOn string
	var depType string

	cmd := &cobra.Command{
		Use:   "add-dependency <job-id-or-slug>",
		Short: "Add a dependency edge to a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(dependsOn) == "" {
				return fmt.Errorf("--depends-on is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			dep, err := cli.AddJobDependency(cmd.Context(), id, client.AddJobDependencyRequest{
				DependsOn: dependsOn,
				Type:      depType,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Added dependency "+styles.Bold.Render(dep.ID)))
				return nil
			}
			return printData(state, dep)
		},
	}

	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "ID or slug of the job this one depends on")
	cmd.Flags().StringVar(&depType, "type", "", "dependency type (success, completion, ...)")

	return cmd
}

func newJobsBatchCommand(state *appState) *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Apply multiple job updates in one call",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(fromFile) == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile) //nolint:gosec // fromFile is from --from-file CLI flag
			if err != nil {
				return fmt.Errorf("read %s: %w", fromFile, err)
			}
			var req client.BatchUpdateJobsRequest
			if err := json.Unmarshal(data, &req); err != nil {
				return fmt.Errorf("invalid batch JSON: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			resp, err := cli.BatchUpdateJobs(cmd.Context(), req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintf(os.Stderr, "%s updated, %d failed\n", styles.Success(fmt.Sprintf("%d", len(resp.Updated))), len(resp.Failed))
				return nil
			}
			return printData(state, resp)
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file containing the batch update payload")
	return cmd
}
