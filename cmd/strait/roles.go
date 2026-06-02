package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newRolesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "Manage RBAC roles",
	}
	cmd.AddCommand(newRolesListCommand(state))
	cmd.AddCommand(newRolesGetCommand(state))
	cmd.AddCommand(newRolesCreateCommand(state))
	cmd.AddCommand(newRolesUpdateCommand(state))
	cmd.AddCommand(newRolesDeleteCommand(state))
	return cmd
}

func newRolesListCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List RBAC roles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListRolesRaw(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRolesGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "get <role-id>",
		Short: "Get a role by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid role id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetRole(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newRolesCreateCommand(state *appState) *cobra.Command {
	var name string
	var description string
	var permissions []string
	var parentRoleID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new RBAC role",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CreateRole(cmd.Context(), name, description, permissions, parentRoleID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created role "+styles.Bold.Render(name)))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "role name (required)")
	cmd.Flags().StringVar(&description, "description", "", "role description")
	cmd.Flags().StringArrayVar(&permissions, "permission", nil, "permission to grant (repeatable)")
	cmd.Flags().StringVar(&parentRoleID, "parent-role-id", "", "parent role ID to inherit from")
	return cmd
}

func newRolesUpdateCommand(state *appState) *cobra.Command {
	var name string
	var description string
	var permissions []string
	var parentRoleID string

	cmd := &cobra.Command{
		Use:   "update <role-id>",
		Short: "Update a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid role id: %w", err)
			}
			if name == "" && description == "" && len(permissions) == 0 && parentRoleID == "" {
				return fmt.Errorf("at least one of --name, --description, --permission, or --parent-role-id must be provided")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.UpdateRole(cmd.Context(), args[0], name, description, permissions, parentRoleID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated role "+styles.Bold.Render(args[0])))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new role name")
	cmd.Flags().StringVar(&description, "description", "", "new role description")
	cmd.Flags().StringArrayVar(&permissions, "permission", nil, "permission to grant (repeatable; replaces existing)")
	cmd.Flags().StringVar(&parentRoleID, "parent-role-id", "", "parent role ID")
	return cmd
}

func newRolesDeleteCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <role-id>",
		Short: "Delete a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid role id: %w", err)
			}
			if err := requireConfirmation(state, "Delete role "+args[0]+"?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteRoleRaw(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted role "+styles.Bold.Render(args[0])))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}
