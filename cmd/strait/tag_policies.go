package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newTagPoliciesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag-policies",
		Short: "Manage RBAC tag policies",
	}
	cmd.AddCommand(newTagPoliciesListCommand(state))
	cmd.AddCommand(newTagPoliciesCreateCommand(state))
	cmd.AddCommand(newTagPoliciesDeleteCommand(state))
	return cmd
}

func newTagPoliciesListCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tag policies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListTagPolicies(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newTagPoliciesCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var resourceType string
	var userID string
	var tagKey string
	var tagValue string
	var actions []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a tag policy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if resourceType == "" {
				return fmt.Errorf("--resource-type is required")
			}
			if userID == "" {
				return fmt.Errorf("--user-id is required")
			}
			if tagKey == "" {
				return fmt.Errorf("--tag-key is required")
			}
			if len(actions) == 0 {
				return fmt.Errorf("at least one --action is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CreateTagPolicy(cmd.Context(), client.TagPolicyRequest{
				ProjectID:    pid,
				ResourceType: resourceType,
				UserID:       userID,
				TagKey:       tagKey,
				TagValue:     tagValue,
				Actions:      actions,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created tag policy"))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "resource type the policy applies to (required)")
	cmd.Flags().StringVar(&userID, "user-id", "", "user ID the policy applies to (required)")
	cmd.Flags().StringVar(&tagKey, "tag-key", "", "tag key to match (required)")
	cmd.Flags().StringVar(&tagValue, "tag-value", "", "tag value to match (optional)")
	cmd.Flags().StringArrayVar(&actions, "action", nil, "action to permit (repeatable; required)")
	return cmd
}

func newTagPoliciesDeleteCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete a tag policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete tag policy "+args[0]+"?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteTagPolicy(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted tag policy "+styles.Bold.Render(args[0])))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}
