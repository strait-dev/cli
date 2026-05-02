package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

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
			from, err := strconv.Atoi(fromV)
			if err != nil || from <= 0 {
				return fmt.Errorf("--from must be a positive integer")
			}
			to, err := strconv.Atoi(toV)
			if err != nil || to <= 0 {
				return fmt.Errorf("--to must be a positive integer")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveWorkflowIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			diff, err := cli.DiffWorkflowVersions(cmd.Context(), id, from, to)
			if err != nil {
				return err
			}
			return printData(state, diff)
		},
	}
	cmd.Flags().StringVar(&fromV, "from", "", "from version (required)")
	cmd.Flags().StringVar(&toV, "to", "", "to version (required)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newWorkflowsPolicyCommand(state *appState) *cobra.Command {
	var setFile string
	var setInline string
	cmd := &cobra.Command{
		Use:   "policy <workflow-id-or-slug>",
		Short: "Get or set the run-time policy for a workflow",
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
			if cmd.Flags().Changed("set") || cmd.Flags().Changed("set-file") {
				policy, err := loadWorkflowPayload(setInline, setFile)
				if err != nil {
					return err
				}
				if policy == nil {
					return fmt.Errorf("--set or --set-file must contain JSON")
				}
				updated, err := cli.SetWorkflowPolicy(cmd.Context(), id, policy)
				if err != nil {
					return err
				}
				return printData(state, updated)
			}
			policy, err := cli.GetWorkflowPolicy(cmd.Context(), id)
			if err != nil {
				return err
			}
			return printData(state, policy)
		},
	}
	cmd.Flags().StringVar(&setInline, "set", "", "inline JSON policy to apply")
	cmd.Flags().StringVar(&setFile, "set-file", "", "path to JSON policy file")
	return cmd
}
