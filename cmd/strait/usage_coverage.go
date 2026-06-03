package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

// registerUsageCoverageCommands attaches additional usage subcommands to the
// existing usage command group. It does not add current, history, or forecast
// (those are registered in usage.go).
func registerUsageCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newUsageAnomaliesCommand(state))
	parent.AddCommand(newUsageProjectsCommand(state))
	parent.AddCommand(newUsageExportCommand(state))
	parent.AddCommand(newUsageEmailPreferencesGetCommand(state))
	parent.AddCommand(newUsageEmailPreferencesSetCommand(state))
}

func newUsageAnomaliesCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "anomalies",
		Short: "Show usage anomalies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetUsageAnomalies(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newUsageProjectsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "projects",
		Short: "Show usage broken down by project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetUsageByProject(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newUsageExportCommand(state *appState) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export usage data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ExportUsage(cmd.Context(), format)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&format, "export-format", "json", "server-side export content format (json, csv)")
	return cmd
}

func newUsageEmailPreferencesGetCommand(state *appState) *cobra.Command {
	var orgID string

	cmd := &cobra.Command{
		Use:   "email-preferences-get",
		Short: "Get usage email preferences for an organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetUsageEmailPreferences(cmd.Context(), orgID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID")
	return cmd
}

func newUsageEmailPreferencesSetCommand(state *appState) *cobra.Command {
	var orgID string
	var monthly bool

	cmd := &cobra.Command{
		Use:   "email-preferences-set",
		Short: "Set usage email preferences for an organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.SetUsageEmailPreferences(cmd.Context(), orgID, monthly)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated usage email preferences"))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID")
	cmd.Flags().BoolVar(&monthly, "monthly", false, "enable monthly usage email")
	return cmd
}
