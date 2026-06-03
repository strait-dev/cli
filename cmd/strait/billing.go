package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newBillingCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Manage billing, budgets, and spending limits",
	}
	cmd.AddCommand(newBillingSpendingLimitCommand(state))
	cmd.AddCommand(newBillingProjectBudgetCommand(state))
	cmd.AddCommand(newBillingAnomalyConfigCommand(state))
	cmd.AddCommand(newBillingRegionsCommand(state))
	cmd.AddCommand(newBillingDowngradePreviewCommand(state))
	cmd.AddCommand(newBillingCheckOrgLimitCommand(state))
	return cmd
}

func newBillingSpendingLimitCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spending-limit",
		Short: "Manage the org-level spending limit",
	}
	cmd.AddCommand(newBillingSpendingLimitGetCommand(state))
	cmd.AddCommand(newBillingSpendingLimitSetCommand(state))
	return cmd
}

func newBillingSpendingLimitGetCommand(state *appState) *cobra.Command {
	var orgID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the org-level spending limit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			orgID, err = requireOrgID(state, orgID)
			if err != nil {
				return err
			}
			out, err := cli.GetSpendingLimit(cmd.Context(), orgID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID (or set STRAIT_ORG / config org)")
	return cmd
}

func newBillingSpendingLimitSetCommand(state *appState) *cobra.Command {
	var orgID string
	var limitMicroUSD int64
	var action string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the org-level spending limit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(action) == "" {
				return fmt.Errorf("--action is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			orgID, err = requireOrgID(state, orgID)
			if err != nil {
				return err
			}
			out, err := cli.SetSpendingLimit(cmd.Context(), orgID, limitMicroUSD, action)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID (or set STRAIT_ORG / config org)")
	cmd.Flags().Int64Var(&limitMicroUSD, "limit-microusd", 0, "spending limit in micro-USD")
	cmd.Flags().StringVar(&action, "action", "", "action to take when limit is reached (required)")
	return cmd
}

func newBillingProjectBudgetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project-budget",
		Short: "Manage the project-level budget",
	}
	cmd.AddCommand(newBillingProjectBudgetGetCommand(state))
	cmd.AddCommand(newBillingProjectBudgetSetCommand(state))
	return cmd
}

func newBillingProjectBudgetGetCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the project-level budget",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetProjectBudget(cmd.Context(), pid)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	return cmd
}

func newBillingProjectBudgetSetCommand(state *appState) *cobra.Command {
	var projectID string
	var budgetMicroUSD int64
	var action string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the project-level budget",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(action) == "" {
				return fmt.Errorf("--action is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.SetProjectBudget(cmd.Context(), pid, budgetMicroUSD, action)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().Int64Var(&budgetMicroUSD, "budget-microusd", 0, "budget in micro-USD")
	cmd.Flags().StringVar(&action, "action", "", "action to take when budget is reached (required)")
	return cmd
}

func newBillingAnomalyConfigCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anomaly-config",
		Short: "Manage the org-level anomaly detection configuration",
	}
	cmd.AddCommand(newBillingAnomalyConfigGetCommand(state))
	cmd.AddCommand(newBillingAnomalyConfigSetCommand(state))
	return cmd
}

func newBillingAnomalyConfigGetCommand(state *appState) *cobra.Command {
	var orgID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the anomaly detection configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			orgID, err = requireOrgID(state, orgID)
			if err != nil {
				return err
			}
			out, err := cli.GetAnomalyConfig(cmd.Context(), orgID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID (or set STRAIT_ORG / config org)")
	return cmd
}

func newBillingAnomalyConfigSetCommand(state *appState) *cobra.Command {
	var orgID string
	var warning float64
	var critical float64
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the anomaly detection thresholds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			orgID, err = requireOrgID(state, orgID)
			if err != nil {
				return err
			}
			out, err := cli.SetAnomalyConfig(cmd.Context(), orgID, warning, critical)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID (or set STRAIT_ORG / config org)")
	cmd.Flags().Float64Var(&warning, "warning", 0, "warning threshold multiplier")
	cmd.Flags().Float64Var(&critical, "critical", 0, "critical threshold multiplier")
	return cmd
}

func newBillingRegionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regions",
		Short: "List available regions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRegions(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	return cmd
}

func newBillingDowngradePreviewCommand(state *appState) *cobra.Command {
	var orgID string
	var targetTier string
	cmd := &cobra.Command{
		Use:   "downgrade-preview",
		Short: "Preview the effects of downgrading an org to a target tier",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(targetTier) == "" {
				return fmt.Errorf("--target-tier is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			orgID, err = requireOrgID(state, orgID)
			if err != nil {
				return err
			}
			out, err := cli.GetDowngradePreview(cmd.Context(), orgID, targetTier)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&orgID, "org", "", "organization ID (or set STRAIT_ORG / config org)")
	cmd.Flags().StringVar(&targetTier, "target-tier", "", "target plan tier to preview (required)")
	return cmd
}

func newBillingCheckOrgLimitCommand(state *appState) *cobra.Command {
	var userID string
	var planTier string
	cmd := &cobra.Command{
		Use:   "check-org-limit",
		Short: "Check whether a user has reached the org limit for a plan tier",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(userID) == "" {
				return fmt.Errorf("--user-id is required")
			}
			if strings.TrimSpace(planTier) == "" {
				return fmt.Errorf("--plan-tier is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CheckOrgLimit(cmd.Context(), userID, planTier)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&userID, "user-id", "", "user ID (required)")
	cmd.Flags().StringVar(&planTier, "plan-tier", "", "plan tier to check (required)")
	return cmd
}
