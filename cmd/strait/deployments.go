package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newDeploymentsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployments",
		Short: "Manage deployment versions and canary rollouts",
	}
	cmd.AddCommand(newDeploymentsListCommand(state))
	cmd.AddCommand(newDeploymentsCreateCommand(state))
	cmd.AddCommand(newDeploymentsActionCommand(state, "finalize", "Finalize a deployment version"))
	cmd.AddCommand(newDeploymentsActionCommand(state, "promote", "Promote a deployment version to active"))
	cmd.AddCommand(newDeploymentsActionCommand(state, "rollback", "Roll back to a previous deployment version"))
	cmd.AddCommand(newDeploymentsCanaryCommand(state))
	return cmd
}

func newDeploymentsListCommand(state *appState) *cobra.Command {
	var environment string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List deployment versions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListDeployments(cmd.Context(), environment, limit)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&environment, "environment", "", "filter by environment")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	return cmd
}

func newDeploymentsCreateCommand(state *appState) *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment version from a JSON manifest file",
		Long:  "Create a deployment version. Provide the full request body (project_id, environment, runtime, artifact_uri, manifest, checksum, strategy, canary_percent, canary_duration) via --from-file.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := readJSONFileBody(fromFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CreateDeployment(cmd.Context(), body)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created deployment version"))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file containing the deployment request body")
	return cmd
}

func newDeploymentsActionCommand(state *appState, action, short string) *cobra.Command {
	var projectID string
	var environment string
	cmd := &cobra.Command{
		Use:   action + " <deployment-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid deployment id: %w", err)
			}
			projectID, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(environment) == "" {
				return fmt.Errorf("--environment is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			var out json.RawMessage
			switch action {
			case "finalize":
				out, err = cli.FinalizeDeployment(cmd.Context(), args[0], projectID, environment)
			case "promote":
				out, err = cli.PromoteDeployment(cmd.Context(), args[0], projectID, environment)
			case "rollback":
				out, err = cli.RollbackDeployment(cmd.Context(), args[0], projectID, environment)
			}
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(action+" complete: "+styles.Bold.Render(args[0])))
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&environment, "environment", "", "environment (required)")
	return cmd
}

func newDeploymentsCanaryCommand(state *appState) *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "canary",
		Short: "Create a canary deployment from a JSON file",
		Long:  "Create a canary deployment. Provide the body (workflow_id, source_version, target_version, traffic_pct, auto_promote) via --from-file.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := readJSONFileBody(fromFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.CreateCanaryDeployment(cmd.Context(), body)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file containing the canary deployment body")
	return cmd
}

// readJSONFileBody reads and validates a JSON request-body file for commands
// that pass a server payload through unchanged.
func readJSONFileBody(path string) (json.RawMessage, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--from-file is required")
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is from a --from-file CLI flag
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("%s does not contain valid JSON", path)
	}
	return json.RawMessage(data), nil
}
