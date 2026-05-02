package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newJobGroupsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "job-groups",
		Aliases: []string{"job-group"},
		Short:   "Manage logical groupings of jobs",
	}

	getCmd := newJobGroupsGetCommand(state)
	getCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	updateCmd := newJobGroupsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	deleteCmd := newJobGroupsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	jobsCmd := newJobGroupsJobsCommand(state)
	jobsCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	pauseCmd := newJobGroupsPauseCommand(state)
	pauseCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	resumeCmd := newJobGroupsResumeCommand(state)
	resumeCmd.ValidArgsFunction = completeJobGroupSlugs(state)
	statsCmd := newJobGroupsStatsCommand(state)
	statsCmd.ValidArgsFunction = completeJobGroupSlugs(state)

	cmd.AddCommand(newJobGroupsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newJobGroupsCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(jobsCmd)
	cmd.AddCommand(pauseCmd)
	cmd.AddCommand(resumeCmd)
	cmd.AddCommand(statsCmd)

	return cmd
}

func newJobGroupsListCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List job groups",
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
			groups, err := cli.ListJobGroups(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(groups))
			for _, g := range groups {
				rows = append(rows, map[string]any{
					"id":        g.ID,
					"name":      g.Name,
					"slug":      g.Slug,
					"paused":    g.Paused,
					"job_count": g.JobCount,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Job Groups", len(groups)))
				for _, g := range groups {
					runState := "running"
					if g.Paused {
						runState = "paused"
					}
					fmt.Fprintf(os.Stderr, "  %-20s  jobs=%d  %s  %s\n",
						styles.Bold.Render(styles.SafeText(g.Slug)), g.JobCount, runState, styles.MutedStyle.Render(styles.SafeText(g.ID)))
				}
				return nil
			}
			return printData(state, rows)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	return cmd
}

func newJobGroupsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <job-group-id-or-slug>",
		Short: "Get job group details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			group, err := cli.GetJobGroup(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", group.ID),
					styles.DetailLine("Name", group.Name),
					styles.DetailLine("Slug", group.Slug),
					styles.DetailLine("Description", group.Description),
					styles.DetailLine("Paused", fmt.Sprintf("%t", group.Paused)),
					styles.DetailLine("Jobs", fmt.Sprintf("%d", group.JobCount)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job Group", lines))
				return nil
			}
			return printData(state, group)
		},
	}
	return cmd
}

func newJobGroupsCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var slug string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a job group",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			if strings.TrimSpace(slug) == "" {
				return fmt.Errorf("--slug is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			group, err := cli.CreateJobGroup(cmd.Context(), client.CreateJobGroupRequest{
				ProjectID:   projectID,
				Name:        name,
				Slug:        slug,
				Description: description,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created job group "+styles.Bold.Render(styles.SafeText(group.Slug))))
				return nil
			}
			return printData(state, group)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "job group name")
	cmd.Flags().StringVar(&slug, "slug", "", "job group slug")
	cmd.Flags().StringVar(&description, "description", "", "job group description")

	return cmd
}

func newJobGroupsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	var description string

	cmd := &cobra.Command{
		Use:   "update <job-group-id-or-slug>",
		Short: "Update a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.UpdateJobGroupRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("slug") {
				req.Slug = &slug
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if req.Name == nil && req.Slug == nil && req.Description == nil {
				return fmt.Errorf("at least one update flag is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			group, err := cli.UpdateJobGroup(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated job group "+styles.Bold.Render(styles.SafeText(group.Slug))))
				return nil
			}
			return printData(state, group)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "job group name")
	cmd.Flags().StringVar(&slug, "slug", "", "job group slug")
	cmd.Flags().StringVar(&description, "description", "", "job group description")

	return cmd
}

func newJobGroupsDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <job-group-id-or-slug>",
		Short: "Delete a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete this job group?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			if err := cli.DeleteJobGroup(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted job group "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": id})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func newJobGroupsJobsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs <job-group-id-or-slug>",
		Short: "List jobs that belong to a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			jobs, err := cli.ListJobsInGroup(cmd.Context(), id)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(jobs))
			for _, j := range jobs {
				rows = append(rows, map[string]any{
					"id":      j.ID,
					"name":    j.Name,
					"slug":    j.Slug,
					"enabled": j.Enabled,
				})
			}
			return printData(state, rows)
		},
	}
	return cmd
}

func newJobGroupsPauseCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause <job-group-id-or-slug>",
		Short: "Pause execution for all jobs in a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			if err := cli.PauseJobGroup(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Paused job group "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"paused": true, "id": id})
		},
	}
	return cmd
}

func newJobGroupsResumeCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <job-group-id-or-slug>",
		Short: "Resume execution for all jobs in a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			if err := cli.ResumeJobGroup(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Resumed job group "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"resumed": true, "id": id})
		},
	}
	return cmd
}

func newJobGroupsStatsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats <job-group-id-or-slug>",
		Short: "Show aggregate metrics for a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobGroupIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			stats, err := cli.GetJobGroupStats(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Group", stats.GroupID),
					styles.DetailLine("Jobs", fmt.Sprintf("%d", stats.JobCount)),
					styles.DetailLine("Runs total", fmt.Sprintf("%d", stats.RunsTotal)),
					styles.DetailLine("Runs failed", fmt.Sprintf("%d", stats.RunsFailed)),
					styles.DetailLine("Runs active", fmt.Sprintf("%d", stats.RunsActive)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job Group Stats", lines))
				return nil
			}
			return printData(state, stats)
		},
	}
	return cmd
}

func resolveJobGroupIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrSlug string) (string, error) {
	if err := validate.SlugOrID(idOrSlug); err != nil {
		return "", fmt.Errorf("invalid job group identifier: %w", err)
	}
	if _, err := cli.GetJobGroup(ctx, idOrSlug); err == nil {
		return idOrSlug, nil
	}
	projectID, err := requireProjectID(state, "")
	if err != nil {
		return "", fmt.Errorf("project is required to resolve slug %q", idOrSlug)
	}
	groups, err := cli.ListJobGroups(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("resolving job group %q: %w", idOrSlug, err)
	}
	for _, g := range groups {
		if g.Slug == idOrSlug {
			return g.ID, nil
		}
	}
	return "", fmt.Errorf("job group %q not found", idOrSlug)
}
