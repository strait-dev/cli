package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newTeamCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage project team members",
	}

	cmd.AddCommand(newTeamListCommand(state))
	cmd.AddCommand(newTeamAddCommand(state))
	cmd.AddCommand(newTeamRemoveCommand(state))
	cmd.AddCommand(newTeamRolesCommand(state))
	cmd.AddCommand(newTeamPoliciesCommand(state))

	return cmd
}

func newTeamPoliciesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policies",
		Short: "Manage RBAC policies on the team",
	}
	cmd.AddCommand(newTeamPoliciesListCommand(state))
	cmd.AddCommand(newTeamPoliciesCreateCommand(state))
	cmd.AddCommand(newTeamPoliciesDeleteCommand(state))
	return cmd
}

func newTeamPoliciesListCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List team RBAC policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			policies, err := cli.ListTeamPolicies(cmd.Context())
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Policies", len(policies)))
				for _, p := range policies {
					fmt.Fprintf(os.Stderr, "  %s  %s  perms=%v\n",
						styles.Bold.Render(p.Name), styles.MutedStyle.Render(p.ID), p.Permissions)
				}
				return nil
			}
			return printData(state, policies)
		},
	}
}

func newTeamPoliciesCreateCommand(state *appState) *cobra.Command {
	var name, resourcePattern, tagPattern string
	var perms []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new team RBAC policy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if len(perms) == 0 {
				return fmt.Errorf("--permission is required at least once")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			policy, err := cli.CreateTeamPolicy(cmd.Context(), client.CreateTeamPolicyRequest{
				Name:            name,
				ResourcePattern: resourcePattern,
				TagPattern:      tagPattern,
				Permissions:     perms,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created policy "+styles.Bold.Render(policy.Name)))
				return nil
			}
			return printData(state, policy)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "policy name (required)")
	cmd.Flags().StringVar(&resourcePattern, "resource-pattern", "", "resource pattern (e.g. job:*, workflow:billing-*)")
	cmd.Flags().StringVar(&tagPattern, "tag-pattern", "", "tag pattern (e.g. env=prod)")
	cmd.Flags().StringSliceVar(&perms, "permission", nil, "permission to grant (repeatable)")
	return cmd
}

func newTeamPoliciesDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete a team RBAC policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, fmt.Sprintf("Delete policy %s?", args[0]), yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteTeamPolicy(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted policy "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]string{"id": args[0], "deleted": "true"})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func newTeamListCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project members",
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

			members, err := cli.ListMembers(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Team Members", len(members)))
				for _, m := range members {
					fmt.Fprintf(os.Stderr, "  %s  role=%s  granted_by=%s\n",
						styles.Bold.Render(m.UserID),
						m.RoleID,
						styles.MutedStyle.Render(m.GrantedBy),
					)
				}
				return nil
			}
			rows := make([]map[string]any, 0, len(members))
			for _, m := range members {
				rows = append(rows, map[string]any{
					"id":         m.ID,
					"project_id": m.ProjectID,
					"user_id":    m.UserID,
					"role_id":    m.RoleID,
					"granted_by": m.GrantedBy,
					"created_at": m.CreatedAt,
				})
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")

	return cmd
}

func newTeamAddCommand(state *appState) *cobra.Command {
	var roleID string

	cmd := &cobra.Command{
		Use:   "add <user-id>",
		Short: "Assign a role to a project member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if roleID == "" {
				return fmt.Errorf("--role-id is required")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			member, err := cli.AddMember(cmd.Context(), client.AssignMemberRequest{
				UserID: args[0],
				RoleID: roleID,
			})
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Added member "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, member)
		},
	}

	cmd.Flags().StringVar(&roleID, "role-id", "", "role ID to assign")

	return cmd
}

func newTeamRemoveCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <user-id>",
		Short: "Remove a member role assignment from the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Remove this member?", yes); err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			if err := cli.RemoveMember(cmd.Context(), args[0]); err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Removed member "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]any{"id": args[0], "removed": true})
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm removal")

	return cmd
}

func newTeamRolesCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "roles",
		Short: "List available roles",
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

			roles, err := cli.ListRoles(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Roles", len(roles)))
				for _, r := range roles {
					perms := fmt.Sprintf("%v", r.Permissions)
					fmt.Fprintf(os.Stderr, "  %s  %s\n",
						styles.Bold.Render(r.Name),
						styles.MutedStyle.Render(perms),
					)
				}
				return nil
			}
			rows := make([]map[string]any, 0, len(roles))
			for _, r := range roles {
				rows = append(rows, map[string]any{
					"id":             r.ID,
					"project_id":     r.ProjectID,
					"name":           r.Name,
					"description":    r.Description,
					"permissions":    r.Permissions,
					"parent_role_id": r.ParentRoleID,
					"is_system":      r.IsSystem,
					"created_at":     r.CreatedAt,
					"updated_at":     r.UpdatedAt,
				})
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")

	return cmd
}
