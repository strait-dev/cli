package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/dag"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newWorkflowsCloneCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	cmd := &cobra.Command{
		Use:   "clone <workflow-id-or-slug>",
		Short: "Clone an existing workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			wf, err := cli.CloneWorkflow(cmd.Context(), id, client.CloneWorkflowRequest{Name: name, Slug: slug})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Cloned to "+styles.Bold.Render(wf.Slug)))
				return nil
			}
			return printData(state, wf)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "name for the cloned workflow")
	cmd.Flags().StringVar(&slug, "slug", "", "slug for the cloned workflow")
	return cmd
}

func loadWorkflowPayload(payload, payloadFile string) (json.RawMessage, error) {
	switch {
	case strings.TrimSpace(payload) != "":
		if !json.Valid([]byte(payload)) {
			return nil, fmt.Errorf("--payload must be valid JSON")
		}
		return json.RawMessage(payload), nil
	case strings.TrimSpace(payloadFile) != "":
		data, err := os.ReadFile(payloadFile) //nolint:gosec // payloadFile is from --payload-file CLI flag
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", payloadFile, err)
		}
		if !json.Valid(data) {
			return nil, fmt.Errorf("payload file %s must be valid JSON", payloadFile)
		}
		return json.RawMessage(data), nil
	}
	return nil, nil
}

func newWorkflowsDryRunCommand(state *appState) *cobra.Command {
	var payload string
	var payloadFile string
	cmd := &cobra.Command{
		Use:   "dry-run <workflow-id-or-slug>",
		Short: "Dry-run a workflow without persisting state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := loadWorkflowPayload(payload, payloadFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			result, err := cli.DryRunWorkflow(cmd.Context(), id, data)
			if err != nil {
				return err
			}
			return printData(state, result)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "inline JSON payload")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "path to payload JSON file")
	return cmd
}

func newWorkflowsPlanCommand(state *appState) *cobra.Command {
	var payload string
	var payloadFile string
	cmd := &cobra.Command{
		Use:   "plan <workflow-id-or-slug>",
		Short: "Show the planned execution graph without running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := loadWorkflowPayload(payload, payloadFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			result, err := cli.PlanWorkflow(cmd.Context(), id, data)
			if err != nil {
				return err
			}
			return printData(state, result)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "inline JSON payload")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "path to payload JSON file")
	return cmd
}

func newWorkflowsSimulateCommand(state *appState) *cobra.Command {
	var payload string
	var payloadFile string
	cmd := &cobra.Command{
		Use:   "simulate <workflow-id-or-slug>",
		Short: "Simulate running a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := loadWorkflowPayload(payload, payloadFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			result, err := cli.SimulateWorkflow(cmd.Context(), id, data)
			if err != nil {
				return err
			}
			return printData(state, result)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "inline JSON payload")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "path to payload JSON file")
	return cmd
}

func newWorkflowsVersionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions <workflow-id-or-slug>",
		Short: "List version history for a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			versions, err := cli.ListWorkflowVersions(cmd.Context(), id)
			if err != nil {
				return err
			}
			return printData(state, versions)
		},
	}
	return cmd
}

func newWorkflowsDiffCommand(state *appState) *cobra.Command {
	var fromV string
	var toV string
	cmd := &cobra.Command{
		Use:   "diff <workflow-id-or-slug>",
		Short: "Show the diff between two workflow versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(fromV) == "" {
				return fmt.Errorf("--from is required (version number or version ID)")
			}
			if strings.TrimSpace(toV) == "" {
				return fmt.Errorf("--to is required (version number or version ID)")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			diff, err := cli.DiffWorkflowVersions(cmd.Context(), id, fromV, toV)
			if err != nil {
				return err
			}
			return printData(state, diff)
		},
	}
	cmd.Flags().StringVar(&fromV, "from", "", "from version number or version ID (required)")
	cmd.Flags().StringVar(&toV, "to", "", "to version number or version ID (required)")
	mustMarkFlagRequired(cmd, "from")
	mustMarkFlagRequired(cmd, "to")
	return cmd
}

func newWorkflowsPolicyCommand(state *appState) *cobra.Command {
	var setFile string
	var setInline string
	var projectID string
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Get or set the workflow governance policy for the project",
		Long: "Get or set the project-level workflow governance policy " +
			"(max fan-out, max depth, forbidden step types, approval requirements).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectID, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("set") || cmd.Flags().Changed("set-file") {
				policy, err := loadWorkflowPayload(setInline, setFile)
				if err != nil {
					return err
				}
				if policy == nil {
					return fmt.Errorf("--set or --set-file must contain JSON")
				}
				updated, err := cli.SetWorkflowPolicy(cmd.Context(), projectID, policy)
				if err != nil {
					return err
				}
				return printData(state, updated)
			}
			policy, err := cli.GetWorkflowPolicy(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			return printData(state, policy)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&setInline, "set", "", "inline JSON policy to apply")
	cmd.Flags().StringVar(&setFile, "set-file", "", "path to JSON policy file")
	return cmd
}

func newWorkflowsVisualizeCommand(state *appState) *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use:   "visualize <workflow-id-or-slug>",
		Short: "Render the workflow DAG",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			workflow, err := cli.GetWorkflow(cmd.Context(), id)
			if err != nil {
				return err
			}

			statusMap := map[string]string(nil)
			if strings.TrimSpace(runID) != "" {
				if err := validate.SlugOrID(runID); err != nil {
					return fmt.Errorf("invalid workflow run id: %w", err)
				}
				stepRuns, err := cli.ListWorkflowStepRuns(cmd.Context(), runID)
				if err != nil {
					return err
				}
				statusMap = make(map[string]string, len(stepRuns))
				for _, stepRun := range stepRuns {
					statusMap[stepRun.StepRef] = string(stepRun.Status)
				}
			}

			steps := make([]dag.Step, 0, len(workflow.Steps))
			for _, step := range workflow.Steps {
				steps = append(steps, dag.Step{
					StepRef:   step.StepRef,
					DependsOn: step.DependsOn,
				})
			}
			rendered := dag.RenderDAG(steps, statusMap)
			payload := map[string]any{
				"workflow_id": workflow.ID,
				"workflow":    workflow,
				"steps":       workflow.Steps,
				"statuses":    statusMap,
				"rendered":    rendered,
			}

			if state.opts.outputFormat == "table" || state.opts.outputFormat == "wide" || (state.opts.outputFormat == "" && isTTYRich(state)) {
				_, err = fmt.Fprintln(state.out(), rendered)
				return err
			}
			return printData(state, payload)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "workflow run ID to overlay step status")
	return cmd
}
