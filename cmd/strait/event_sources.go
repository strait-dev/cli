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

func newEventSourcesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "event-sources",
		Aliases: []string{"event-source"},
		Short:   "Manage external event sources",
	}

	getCmd := newEventSourcesGetCommand(state)
	getCmd.ValidArgsFunction = completeEventSourceSlugs(state)
	updateCmd := newEventSourcesUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeEventSourceSlugs(state)
	deleteCmd := newEventSourcesDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeEventSourceSlugs(state)

	cmd.AddCommand(newEventSourcesListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newEventSourcesCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)

	return cmd
}

func newEventSourcesListCommand(state *appState) *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List event sources",
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
			sources, err := cli.ListEventSources(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(sources))
			for _, s := range sources {
				rows = append(rows, map[string]any{
					"id":      s.ID,
					"name":    s.Name,
					"slug":    s.Slug,
					"type":    s.Type,
					"enabled": s.Enabled,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Event Sources", len(sources)))
				for _, s := range sources {
					fmt.Fprintf(os.Stderr, "  %s  %-20s  type=%s  %s\n",
						styles.Enabled(s.Enabled),
						styles.Bold.Render(s.Slug),
						s.Type,
						styles.MutedStyle.Render(s.ID),
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

func newEventSourcesGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <event-source-id-or-slug>",
		Short: "Get event source details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEventSourceIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			source, err := cli.GetEventSource(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", source.ID),
					styles.DetailLine("Name", source.Name),
					styles.DetailLine("Slug", source.Slug),
					styles.DetailLine("Type", source.Type),
					styles.DetailLine("Enabled", styles.Enabled(source.Enabled)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Event Source", lines))
				return nil
			}
			return printData(state, source)
		},
	}
	return cmd
}

func newEventSourcesCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var name string
	var slug string
	var sourceType string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an event source",
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
			if strings.TrimSpace(sourceType) == "" {
				return fmt.Errorf("--type is required")
			}
			req := client.CreateEventSourceRequest{
				ProjectID: projectID,
				Name:      name,
				Slug:      slug,
				Type:      sourceType,
			}
			if strings.TrimSpace(configJSON) != "" {
				if !json.Valid([]byte(configJSON)) {
					return fmt.Errorf("--config-json must be valid JSON")
				}
				req.Config = json.RawMessage(configJSON)
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			source, err := cli.CreateEventSource(cmd.Context(), req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created event source "+styles.Bold.Render(source.Slug)))
				return nil
			}
			return printData(state, source)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "event source name")
	cmd.Flags().StringVar(&slug, "slug", "", "event source slug")
	cmd.Flags().StringVar(&sourceType, "type", "", "event source type (kafka, pubsub, sqs, etc.)")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "JSON-encoded source configuration")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the source is enabled")

	return cmd
}

func newEventSourcesUpdateCommand(state *appState) *cobra.Command {
	var name string
	var slug string
	var configJSON string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "update <event-source-id-or-slug>",
		Short: "Update an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.UpdateEventSourceRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("slug") {
				req.Slug = &slug
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
			if req.Name == nil && req.Slug == nil && req.Config == nil && req.Enabled == nil {
				return fmt.Errorf("at least one update flag is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEventSourceIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			source, err := cli.UpdateEventSource(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated event source "+styles.Bold.Render(source.Slug)))
				return nil
			}
			return printData(state, source)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "event source name")
	cmd.Flags().StringVar(&slug, "slug", "", "event source slug")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "JSON-encoded source configuration")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the source is enabled")

	return cmd
}

func newEventSourcesDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <event-source-id-or-slug>",
		Short: "Delete an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete this event source?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveEventSourceIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			if err := cli.DeleteEventSource(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted event source "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": id})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func resolveEventSourceIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrSlug string) (string, error) {
	if err := validate.SlugOrID(idOrSlug); err != nil {
		return "", fmt.Errorf("invalid event source identifier: %w", err)
	}
	if _, err := cli.GetEventSource(ctx, idOrSlug); err == nil {
		return idOrSlug, nil
	}
	projectID, err := requireProjectID(state, "")
	if err != nil {
		return "", fmt.Errorf("project is required to resolve slug %q", idOrSlug)
	}
	sources, err := cli.ListEventSources(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("resolving event source %q: %w", idOrSlug, err)
	}
	for _, s := range sources {
		if s.Slug == idOrSlug {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("event source %q not found", idOrSlug)
}
