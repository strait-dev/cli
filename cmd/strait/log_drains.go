package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newLogDrainsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log-drains",
		Aliases: []string{"log-drain"},
		Short:   "Manage log drain destinations",
	}

	getCmd := newLogDrainsGetCommand(state)
	getCmd.ValidArgsFunction = completeLogDrainIDs(state)
	updateCmd := newLogDrainsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeLogDrainIDs(state)
	deleteCmd := newLogDrainsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeLogDrainIDs(state)

	cmd.AddCommand(newLogDrainsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newLogDrainsCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)

	return cmd
}

func newLogDrainsListCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List log drains",
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
			drains, err := cli.ListLogDrains(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(drains))
			for _, d := range drains {
				rows = append(rows, map[string]any{
					"id":      d.ID,
					"name":    d.Name,
					"type":    d.Type,
					"enabled": d.Enabled,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Log Drains", len(drains)))
				for _, d := range drains {
					fmt.Fprintf(os.Stderr, "  %s  %-20s  type=%s  %s\n",
						styles.Enabled(d.Enabled),
						styles.Bold.Render(styles.SafeText(d.Name)),
						styles.SafeText(d.Type),
						styles.MutedStyle.Render(styles.SafeText(d.ID)),
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

func newLogDrainsGetCommand(state *appState) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "get <drain-id>",
		Short: "Get log drain details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid log drain id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			drain, err := cli.GetLogDrain(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", drain.ID),
					styles.DetailLine("Name", drain.Name),
					styles.DetailLine("Type", drain.Type),
					styles.DetailLine("Enabled", styles.Enabled(drain.Enabled)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Log Drain", lines))
				return nil
			}
			masked := *drain
			masked.Config = maskRawJSON(drain.Config, reveal)
			return printData(state, &masked)
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "show drain config (API keys, URLs) in plaintext")
	return cmd
}

func newLogDrainsCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var drainType string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a log drain",
		Long: `Creates a log drain of type datadog, splunk, http, or other supported types.
Pass drain-specific settings via --config-json.`,
		Example: `  strait log-drains create --type datadog --name prod-dd --config-json '{"api_key":"xxx","site":"us"}'
  strait log-drains create --type http --name siem --config-json '{"url":"https://siem.example.com/ingest"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			if strings.TrimSpace(drainType) == "" {
				return fmt.Errorf("--type is required")
			}
			if strings.TrimSpace(configJSON) == "" {
				return fmt.Errorf("--config-json is required")
			}
			if !json.Valid([]byte(configJSON)) {
				return fmt.Errorf("--config-json must be valid JSON")
			}
			req := client.CreateLogDrainRequest{
				ProjectID: projectID,
				Name:      name,
				Type:      drainType,
				Config:    json.RawMessage(configJSON),
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			drain, err := cli.CreateLogDrain(cmd.Context(), req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created log drain "+styles.Bold.Render(styles.SafeText(drain.Name))))
				return nil
			}
			masked := *drain
			masked.Config = maskRawJSON(drain.Config, false)
			return printData(state, &masked)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "drain name")
	cmd.Flags().StringVar(&drainType, "type", "", "drain type (datadog, splunk, http, ...)")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "drain config as JSON")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the drain is enabled")

	return cmd
}

func newLogDrainsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "update <drain-id>",
		Short: "Update a log drain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid log drain id: %w", err)
			}
			req := client.UpdateLogDrainRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("config-json") {
				if strings.TrimSpace(configJSON) != "" && !json.Valid([]byte(configJSON)) {
					return fmt.Errorf("--config-json must be valid JSON")
				}
				raw := json.RawMessage(configJSON)
				req.Config = &raw
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			if req.Name == nil && req.Config == nil && req.Enabled == nil {
				return fmt.Errorf("at least one update flag is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			drain, err := cli.UpdateLogDrain(cmd.Context(), args[0], req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated log drain "+styles.Bold.Render(styles.SafeText(drain.Name))))
				return nil
			}
			masked := *drain
			masked.Config = maskRawJSON(drain.Config, false)
			return printData(state, &masked)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "drain name")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "drain config as JSON")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the drain is enabled")

	return cmd
}

func newLogDrainsDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <drain-id>",
		Short: "Delete a log drain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid log drain id: %w", err)
			}
			if err := requireConfirmation(state, "Delete this log drain?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteLogDrain(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted log drain "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": args[0]})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}
