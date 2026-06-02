package main

import (
	"fmt"

	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newOrganizationsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organizations",
		Short: "Access organization-scoped resources",
	}
	cmd.AddCommand(newOrganizationsJobsCommand(state))
	cmd.AddCommand(newOrganizationsRunsCommand(state))
	return cmd
}

func newOrganizationsJobsCommand(state *appState) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "jobs <org-id>",
		Short: "List jobs for an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid organization id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListOrgJobs(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	return cmd
}

func newOrganizationsRunsCommand(state *appState) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "runs <org-id>",
		Short: "List runs for an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid organization id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListOrgRuns(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	return cmd
}
