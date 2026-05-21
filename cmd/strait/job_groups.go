package main

import (
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
		Short:   "Manage job groups",
	}

	getCmd := newJobGroupsGetCommand(state)
	getCmd.ValidArgsFunction = completeJobGroupIDs(state)
	updateCmd := newJobGroupsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeJobGroupIDs(state)
	deleteCmd := newJobGroupsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeJobGroupIDs(state)
	jobsCmd := newJobGroupsJobsCommand(state)
	jobsCmd.ValidArgsFunction = completeJobGroupIDs(state)
	pauseCmd := newJobGroupsPauseCommand(state)
	pauseCmd.ValidArgsFunction = completeJobGroupIDs(state)
	resumeCmd := newJobGroupsResumeCommand(state)
	resumeCmd.ValidArgsFunction = completeJobGroupIDs(state)
	statsCmd := newJobGroupsStatsCommand(state)
	statsCmd.ValidArgsFunction = completeJobGroupIDs(state)

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
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			groups, err := cli.ListJobGroups(cmd.Context(), pid)
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
					fmt.Fprintf(os.Stderr, "  %s  %-20s  jobs=%d  %s\n",
						styles.Enabled(!g.Paused),
						styles.Bold.Render(styles.SafeText(g.Slug)),
						g.JobCount,
						styles.MutedStyle.Render(styles.SafeText(g.ID)),
					)
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
	return &cobra.Command{
		Use:   "get <group-id>",
		Short: "Get job group details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			group, err := cli.GetJobGroup(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", group.ID),
					styles.DetailLine("Name", group.Name),
					styles.DetailLine("Slug", group.Slug),
					styles.DetailLine("Paused", fmt.Sprintf("%t", group.Paused)),
					styles.DetailLine("Jobs", fmt.Sprintf("%d", group.JobCount)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job Group", lines))
				return nil
			}
			return printData(state, group)
		},
	}
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
			pid, err := requireProjectID(state, projectID)
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
				ProjectID:   pid,
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
	cmd.Flags().StringVar(&name, "name", "", "group name")
	cmd.Flags().StringVar(&slug, "slug", "", "group slug")
	cmd.Flags().StringVar(&description, "description", "", "group description")
	return cmd
}

func newJobGroupsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	var description string
	cmd := &cobra.Command{
		Use:   "update <group-id>",
		Short: "Update a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
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
			group, err := cli.UpdateJobGroup(cmd.Context(), args[0], req)
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
	cmd.Flags().StringVar(&name, "name", "", "group name")
	cmd.Flags().StringVar(&slug, "slug", "", "group slug")
	cmd.Flags().StringVar(&description, "description", "", "group description")
	return cmd
}

func newJobGroupsDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <group-id>",
		Short: "Delete a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
			if err := requireConfirmation(state, "Delete this job group?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteJobGroup(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted job group "+styles.Bold.Render(styles.SafeText(args[0]))))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": args[0]})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func newJobGroupsJobsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "jobs <group-id>",
		Short: "List jobs in a job group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			jobs, err := cli.ListJobsInGroup(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, jobs)
		},
	}
}

func newJobGroupsPauseCommand(state *appState) *cobra.Command {
	return newJobGroupStateCommand(state, "pause <group-id>", "Pause all jobs in a job group", "paused")
}

func newJobGroupsResumeCommand(state *appState) *cobra.Command {
	return newJobGroupStateCommand(state, "resume <group-id>", "Resume all jobs in a job group", "resumed")
}

func newJobGroupStateCommand(state *appState, use, short, status string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if status == "paused" {
				err = cli.PauseJobGroup(cmd.Context(), args[0])
			} else {
				err = cli.ResumeJobGroup(cmd.Context(), args[0])
			}
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				verb := "Paused"
				if status == "resumed" {
					verb = "Resumed"
				}
				fmt.Fprintln(os.Stderr, styles.Success(verb+" job group "+styles.Bold.Render(styles.SafeText(args[0]))))
				return nil
			}
			return printData(state, map[string]string{"id": args[0], "status": status})
		},
	}
}

func newJobGroupsStatsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "stats <group-id>",
		Short: "Show job group statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job group id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			stats, err := cli.GetJobGroupStats(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, stats)
		},
	}
}
