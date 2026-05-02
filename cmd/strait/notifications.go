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

func newNotificationsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notifications",
		Aliases: []string{"notification"},
		Short:   "Manage notification channels",
	}

	getCmd := newNotificationsGetCommand(state)
	getCmd.ValidArgsFunction = completeNotificationChannelIDs(state)
	updateCmd := newNotificationsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeNotificationChannelIDs(state)
	deleteCmd := newNotificationsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeNotificationChannelIDs(state)

	cmd.AddCommand(newNotificationsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newNotificationsCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)

	return cmd
}

func newNotificationsListCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notification channels",
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
			channels, err := cli.ListNotificationChannels(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(channels))
			for _, c := range channels {
				rows = append(rows, map[string]any{
					"id":      c.ID,
					"name":    c.Name,
					"type":    c.Type,
					"enabled": c.Enabled,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Notification Channels", len(channels)))
				for _, c := range channels {
					fmt.Fprintf(os.Stderr, "  %s  %-20s  type=%s  %s\n",
						styles.Enabled(c.Enabled),
						styles.Bold.Render(c.Name),
						c.Type,
						styles.MutedStyle.Render(c.ID),
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

func newNotificationsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <channel-id>",
		Short: "Get notification channel details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid notification channel id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			channel, err := cli.GetNotificationChannel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", channel.ID),
					styles.DetailLine("Name", channel.Name),
					styles.DetailLine("Type", channel.Type),
					styles.DetailLine("Enabled", styles.Enabled(channel.Enabled)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Notification Channel", lines))
				return nil
			}
			return printData(state, channel)
		},
	}
	return cmd
}

func newNotificationsCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var channelType string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a notification channel",
		Long: `Creates a notification channel of type slack, email, pagerduty, or other supported types.
Pass channel-specific settings via --config-json (e.g. webhook URL for slack, email address for email).`,
		Example: `  strait notifications create --type slack --name oncall --config-json '{"webhook_url":"https://hooks.slack.com/..."}'
  strait notifications create --type email --name release --config-json '{"to":"team@example.com"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			if strings.TrimSpace(channelType) == "" {
				return fmt.Errorf("--type is required")
			}
			if strings.TrimSpace(configJSON) == "" {
				return fmt.Errorf("--config-json is required")
			}
			if !json.Valid([]byte(configJSON)) {
				return fmt.Errorf("--config-json must be valid JSON")
			}
			req := client.CreateNotificationChannelRequest{
				ProjectID: projectID,
				Name:      name,
				Type:      channelType,
				Config:    json.RawMessage(configJSON),
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			channel, err := cli.CreateNotificationChannel(cmd.Context(), req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created channel "+styles.Bold.Render(channel.Name)))
				return nil
			}
			return printData(state, channel)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "channel name")
	cmd.Flags().StringVar(&channelType, "type", "", "channel type (slack, email, pagerduty, ...)")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "channel config as JSON")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the channel is enabled")

	return cmd
}

func newNotificationsUpdateCommand(state *appState) *cobra.Command {
	var name string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "update <channel-id>",
		Short: "Update a notification channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid notification channel id: %w", err)
			}
			req := client.UpdateNotificationChannelRequest{}
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
			channel, err := cli.UpdateNotificationChannel(cmd.Context(), args[0], req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated channel "+styles.Bold.Render(channel.Name)))
				return nil
			}
			return printData(state, channel)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "channel name")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "channel config as JSON")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the channel is enabled")

	return cmd
}

func newNotificationsDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <channel-id>",
		Short: "Delete a notification channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid notification channel id: %w", err)
			}
			if err := requireConfirmation(state, "Delete this notification channel?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteNotificationChannel(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted channel "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": args[0]})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}
