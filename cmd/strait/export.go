package main

import (
	"github.com/spf13/cobra"
)

func newExportCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export jobs, runs, and workflows data",
	}
	cmd.AddCommand(newExportJobsCommand(state))
	cmd.AddCommand(newExportRunsCommand(state))
	cmd.AddCommand(newExportWorkflowsCommand(state))
	return cmd
}

func newExportJobsCommand(state *appState) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Export jobs data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ExportJobs(cmd.Context(), format)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&format, "export-format", "json", "server-side export content format (json, csv)")
	return cmd
}

func newExportRunsCommand(state *appState) *cobra.Command {
	var format string
	var from string
	var to string
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Export runs data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ExportRuns(cmd.Context(), format, from, to)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&format, "export-format", "json", "server-side export content format (json, csv)")
	cmd.Flags().StringVar(&from, "from", "", "start of time range (RFC3339)")
	cmd.Flags().StringVar(&to, "to", "", "end of time range (RFC3339)")
	return cmd
}

func newExportWorkflowsCommand(state *appState) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "workflows",
		Short: "Export workflows data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ExportWorkflows(cmd.Context(), format)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&format, "export-format", "json", "server-side export content format (json, csv)")
	return cmd
}
