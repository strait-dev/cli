package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newWorkflowsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflows",
		Short: "Manage workflows",
		Long:  idOrSlugLong("workflow", "Manage workflows."),
	}

	getCmd := newWorkflowsGetCommand(state)
	getCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	updateCmd := newWorkflowsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	deleteCmd := newWorkflowsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	triggerCmd := newWorkflowsTriggerCommand(state)
	triggerCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	cloneCmd := newWorkflowsCloneCommand(state)
	cloneCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	dryRunCmd := newWorkflowsDryRunCommand(state)
	dryRunCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	planCmd := newWorkflowsPlanCommand(state)
	planCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	simulateCmd := newWorkflowsSimulateCommand(state)
	simulateCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	versionsCmd := newWorkflowsVersionsCommand(state)
	versionsCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	diffCmd := newWorkflowsDiffCommand(state)
	diffCmd.ValidArgsFunction = completeWorkflowSlugs(state)
	policyCmd := newWorkflowsPolicyCommand(state)
	policyCmd.ValidArgsFunction = completeWorkflowSlugs(state)

	cmd.AddCommand(newWorkflowsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newWorkflowsCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(triggerCmd)
	cmd.AddCommand(cloneCmd)
	cmd.AddCommand(dryRunCmd)
	cmd.AddCommand(planCmd)
	cmd.AddCommand(simulateCmd)
	cmd.AddCommand(versionsCmd)
	cmd.AddCommand(diffCmd)
	cmd.AddCommand(policyCmd)

	return cmd
}

func newWorkflowsListCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflows",
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
			workflows, err := cli.ListWorkflows(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(workflows))
			for _, wf := range workflows {
				rows = append(rows, map[string]any{
					"id":      wf.ID,
					"name":    wf.Name,
					"slug":    wf.Slug,
					"enabled": styles.Enabled(wf.Enabled),
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Workflows", len(workflows)))
				for _, wf := range workflows {
					fmt.Fprintf(os.Stderr, "  %s  %-20s  %s\n",
						styles.Enabled(wf.Enabled),
						styles.Bold.Render(wf.Slug),
						styles.MutedStyle.Render(wf.ID),
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

func newWorkflowsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <workflow-id-or-slug>",
		Short: "Get workflow by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			wf, err := cli.GetWorkflow(cmd.Context(), workflowID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", wf.ID),
					styles.DetailLine("Name", wf.Name),
					styles.DetailLine("Slug", wf.Slug),
					styles.DetailLine("Enabled", styles.Enabled(wf.Enabled)),
					styles.DetailLine("Version", fmt.Sprintf("%d", wf.Version)),
					styles.DetailLine("Steps", fmt.Sprintf("%d", len(wf.Steps))),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Workflow", lines))
				return nil
			}
			return printData(state, wf)
		},
	}

	return cmd
}

func newWorkflowsCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var slug string
	var description string
	var stepsJSON string
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create workflow",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if projectID == "" {
				projectID = state.opts.projectID
			}
			if projectID == "" || name == "" || slug == "" {
				return fmt.Errorf("project, name, and slug are required")
			}

			req := client.CreateWorkflowRequest{
				ProjectID:   projectID,
				Name:        name,
				Slug:        slug,
				Description: description,
			}
			if strings.TrimSpace(stepsJSON) != "" {
				if err := json.Unmarshal([]byte(stepsJSON), &req.Steps); err != nil {
					return fmt.Errorf("invalid --steps-json: %w", err)
				}
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			wf, err := cli.CreateWorkflow(cmd.Context(), req, idempotencyKey)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created workflow "+styles.Bold.Render(wf.Slug)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("ID", wf.ID))
				return nil
			}
			return printData(state, wf)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "workflow name")
	cmd.Flags().StringVar(&slug, "slug", "", "workflow slug")
	cmd.Flags().StringVar(&description, "description", "", "workflow description")
	cmd.Flags().StringVar(&stepsJSON, "steps-json", "", "JSON array of workflow steps")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "idempotency key to prevent duplicate creates (passed as X-Idempotency-Key header)")

	return cmd
}

func newWorkflowsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	var description string
	var enabled bool
	var stepsJSON string

	cmd := &cobra.Command{
		Use:   "update <workflow-id-or-slug>",
		Short: "Update a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.UpdateWorkflowRequest{}

			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("slug") {
				req.Slug = &slug
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			if cmd.Flags().Changed("steps-json") {
				steps := make([]client.WorkflowStepRequest, 0)
				if strings.TrimSpace(stepsJSON) != "" {
					if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
						return fmt.Errorf("invalid --steps-json: %w", err)
					}
				}
				req.Steps = &steps
			}

			if req.Name == nil && req.Slug == nil && req.Description == nil && req.Enabled == nil && req.Steps == nil {
				return fmt.Errorf("at least one update flag is required")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			wf, err := cli.UpdateWorkflow(cmd.Context(), workflowID, req)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated workflow "+styles.Bold.Render(wf.Slug)))
				return nil
			}
			return printData(state, wf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "workflow name")
	cmd.Flags().StringVar(&slug, "slug", "", "workflow slug")
	cmd.Flags().StringVar(&description, "description", "", "workflow description")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "workflow enabled state")
	cmd.Flags().StringVar(&stepsJSON, "steps-json", "", "JSON array of workflow steps (set empty string to clear)")

	return cmd
}

func newWorkflowsDeleteCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <workflow-id-or-slug>",
		Short: "Delete a workflow by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete this workflow?", yes); err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			if err := cli.DeleteWorkflow(cmd.Context(), workflowID); err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted workflow "+styles.Bold.Render(workflowID)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": workflowID})
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")

	return cmd
}

func newWorkflowsTriggerCommand(state *appState) *cobra.Command {
	var payload string
	var payloadFile string
	var projectID string

	cmd := &cobra.Command{
		Use:   "trigger <workflow-id-or-slug>",
		Short: "Trigger workflow by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			workflowID, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			req := client.TriggerWorkflowRequest{ProjectID: projectID}
			if payloadFile != "" {
				raw, err := os.ReadFile(payloadFile) //nolint:gosec // payloadFile is from --payload-file CLI flag
				if err != nil {
					return err
				}
				req.Payload = json.RawMessage(raw)
			} else if strings.TrimSpace(payload) != "" {
				req.Payload = json.RawMessage(payload)
			}
			if len(req.Payload) > 0 && !json.Valid(req.Payload) {
				return fmt.Errorf("payload must be valid JSON")
			}

			run, err := cli.TriggerWorkflow(cmd.Context(), workflowID, req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info("Triggered workflow run "+styles.Bold.Render(run.ID)))
				return nil
			}
			return printData(state, run)
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "inline JSON payload")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "path to payload JSON file")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")

	return cmd
}

func resolveWorkflowIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrSlug string) (string, error) {
	if err := validate.SlugOrID(idOrSlug); err != nil {
		return "", fmt.Errorf("invalid workflow identifier: %w", err)
	}
	if validate.IsUUID(idOrSlug) {
		return idOrSlug, nil
	}
	_, err := cli.GetWorkflow(ctx, idOrSlug)
	if err == nil {
		return idOrSlug, nil
	}
	if !client.IsNotFound(err) {
		return "", fmt.Errorf("resolving workflow %q: %w", idOrSlug, err)
	}

	projectID, perr := requireProjectID(state, "")
	if perr != nil {
		return "", fmt.Errorf("project is required to resolve slug %q", idOrSlug)
	}

	workflows, lerr := cli.ListWorkflows(ctx, projectID)
	if lerr != nil {
		return "", fmt.Errorf("resolving workflow %q: %w", idOrSlug, lerr)
	}

	for _, workflow := range workflows {
		if workflow.Slug == idOrSlug {
			return workflow.ID, nil
		}
	}

	return "", fmt.Errorf("workflow %q not found", idOrSlug)
}
