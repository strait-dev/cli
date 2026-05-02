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

func newEnvironmentsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"environment", "envs", "env"},
		Short:   "Manage project environments",
	}

	getCmd := newEnvironmentsGetCommand(state)
	getCmd.ValidArgsFunction = completeEnvironmentSlugs(state)
	updateCmd := newEnvironmentsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeEnvironmentSlugs(state)
	deleteCmd := newEnvironmentsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeEnvironmentSlugs(state)
	variablesCmd := newEnvironmentsVariablesCommand(state)
	variablesCmd.ValidArgsFunction = completeEnvironmentSlugs(state)

	cmd.AddCommand(newEnvironmentsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newEnvironmentsCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(variablesCmd)

	return cmd
}

func newEnvironmentsListCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
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
			envs, err := cli.ListEnvironments(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(envs))
			for _, e := range envs {
				rows = append(rows, map[string]any{
					"id":          e.ID,
					"name":        e.Name,
					"slug":        e.Slug,
					"is_standard": e.IsStandard,
					"variables":   len(e.Variables),
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Environments", len(envs)))
				for _, e := range envs {
					fmt.Fprintf(os.Stderr, "  %-20s  vars=%d  %s\n",
						styles.Bold.Render(e.Slug),
						len(e.Variables),
						styles.MutedStyle.Render(e.ID),
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

func newEnvironmentsGetCommand(state *appState) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "get <environment-id-or-slug>",
		Short: "Get environment by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEnvironmentIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			env, err := cli.GetEnvironment(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", env.ID),
					styles.DetailLine("Name", env.Name),
					styles.DetailLine("Slug", env.Slug),
					styles.DetailLine("Standard", fmt.Sprintf("%t", env.IsStandard)),
					styles.DetailLine("Variables", fmt.Sprintf("%d", len(env.Variables))),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Environment Details", lines))
				return nil
			}
			masked := *env
			masked.Variables = maskMapValues(env.Variables, reveal)
			return printData(state, &masked)
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "show variable values in plaintext (default: masked as ********)")
	return cmd
}

func newEnvironmentsCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var slug string
	var parentID string
	var isStandard bool
	var variables []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new environment",
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

			vars, err := parseKeyValuePairs(variables)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			env, err := cli.CreateEnvironment(cmd.Context(), client.CreateEnvironmentRequest{
				ProjectID:  projectID,
				Name:       name,
				Slug:       slug,
				ParentID:   parentID,
				IsStandard: isStandard,
				Variables:  vars,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created environment "+styles.Bold.Render(env.Slug)))
				return nil
			}
			return printData(state, env)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "environment name")
	cmd.Flags().StringVar(&slug, "slug", "", "environment slug")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "parent environment ID for inheritance")
	cmd.Flags().BoolVar(&isStandard, "standard", false, "mark as a standard environment")
	cmd.Flags().StringArrayVar(&variables, "var", nil, "environment variable as KEY=VALUE (repeatable)")

	return cmd
}

func newEnvironmentsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	var parentID string
	var variables []string

	cmd := &cobra.Command{
		Use:   "update <environment-id-or-slug>",
		Short: "Update an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.UpdateEnvironmentRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("slug") {
				req.Slug = &slug
			}
			if cmd.Flags().Changed("parent-id") {
				req.ParentID = &parentID
			}
			if cmd.Flags().Changed("var") {
				vars, err := parseKeyValuePairs(variables)
				if err != nil {
					return err
				}
				req.Variables = &vars
			}
			if req.Name == nil && req.Slug == nil && req.ParentID == nil && req.Variables == nil {
				return fmt.Errorf("at least one update flag is required")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEnvironmentIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			env, err := cli.UpdateEnvironment(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated environment "+styles.Bold.Render(env.Slug)))
				return nil
			}
			return printData(state, env)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "environment name")
	cmd.Flags().StringVar(&slug, "slug", "", "environment slug")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "parent environment ID")
	cmd.Flags().StringArrayVar(&variables, "var", nil, "replace variables map; KEY=VALUE (repeatable)")

	return cmd
}

func newEnvironmentsDeleteCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <environment-id-or-slug>",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete this environment?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEnvironmentIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			if err := cli.DeleteEnvironment(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted environment "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": id})
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")

	return cmd
}

func newEnvironmentsVariablesCommand(state *appState) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "variables <environment-id-or-slug>",
		Short: "List environment variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEnvironmentIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			vars, err := cli.ListEnvironmentVariables(cmd.Context(), id)
			if err != nil {
				return err
			}
			masked := maskMapValues(vars, reveal)
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Variables", len(masked)))
				for k, v := range masked {
					fmt.Fprintf(os.Stderr, "  %s=%s\n", styles.Bold.Render(k), v)
				}
				return nil
			}
			return printData(state, masked)
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "show variable values in plaintext (default: masked as ********)")
	return cmd
}

func resolveEnvironmentIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrSlug string) (string, error) {
	if err := validate.SlugOrID(idOrSlug); err != nil {
		return "", fmt.Errorf("invalid environment identifier: %w", err)
	}
	if _, err := cli.GetEnvironment(ctx, idOrSlug); err == nil {
		return idOrSlug, nil
	}
	projectID, err := requireProjectID(state, "")
	if err != nil {
		return "", fmt.Errorf("project is required to resolve slug %q", idOrSlug)
	}
	envs, err := cli.ListEnvironments(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("resolving environment %q: %w", idOrSlug, err)
	}
	for _, e := range envs {
		if e.Slug == idOrSlug {
			return e.ID, nil
		}
	}
	return "", fmt.Errorf("environment %q not found", idOrSlug)
}

func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("invalid key=value (missing %q): %q", "=", p)
		}
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, fmt.Errorf("invalid key=value (empty key): %q", p)
		}
		if err := validateVarKey(k); err != nil {
			return nil, fmt.Errorf("invalid key %q: %w", k, err)
		}
		if _, dup := out[k]; dup {
			return nil, fmt.Errorf("duplicate key %q", k)
		}
		out[k] = v
	}
	return out, nil
}

func validateVarKey(k string) error {
	for _, r := range k {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("key must not contain control characters")
		}
	}
	return nil
}
