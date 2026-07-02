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
		Long:    "Manage external event sources.",
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
					"id":                  s.ID,
					"name":                s.Name,
					"description":         s.Description,
					"enabled":             s.Enabled,
					"signature_algorithm": s.SignatureAlgorithm,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Event Sources", len(sources)))
				for _, s := range sources {
					fmt.Fprintf(os.Stderr, "  %s  %-24s  signature=%s  %s\n",
						styles.Enabled(s.Enabled),
						styles.Bold.Render(styles.SafeText(s.Name)),
						styles.SafeText(s.SignatureAlgorithm),
						styles.MutedStyle.Render(styles.SafeText(s.ID)),
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
		Use:   "get <event-source-id>",
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
					styles.DetailLine("Description", source.Description),
					styles.DetailLine("Signature", source.SignatureAlgorithm),
					styles.DetailLine("Signature Header", source.SignatureHeader),
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
	var description string
	var configJSON string
	var schemaJSON string
	var signatureHeader string
	var signatureAlgorithm string
	var signatureSecret string
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
			req := client.CreateEventSourceRequest{
				ProjectID:          projectID,
				Name:               name,
				Description:        description,
				SignatureHeader:    signatureHeader,
				SignatureAlgorithm: signatureAlgorithm,
				SignatureSecret:    signatureSecret,
			}
			rawSchema := strings.TrimSpace(schemaJSON)
			if rawSchema == "" {
				rawSchema = strings.TrimSpace(configJSON)
			}
			if rawSchema != "" {
				if !json.Valid([]byte(rawSchema)) {
					return fmt.Errorf("--schema-json must be valid JSON")
				}
				req.Schema = json.RawMessage(rawSchema)
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
				fmt.Fprintln(os.Stderr, styles.Success("Created event source "+styles.Bold.Render(styles.SafeText(source.ID))))
				return nil
			}
			return printData(state, source)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&name, "name", "", "event source name")
	cmd.Flags().StringVar(&description, "description", "", "event source description")
	cmd.Flags().StringVar(&schemaJSON, "schema-json", "", "JSON-encoded event payload schema")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "deprecated alias for --schema-json")
	cmd.Flags().StringVar(&signatureHeader, "signature-header", "", "header containing the event signature")
	cmd.Flags().StringVar(&signatureAlgorithm, "signature-algorithm", "", "signature algorithm (hmac-sha256, stripe-v1, github-sha256)")
	cmd.Flags().StringVar(&signatureSecret, "signature-secret", "", "secret used to verify event signatures")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the source is enabled")

	return cmd
}

func newEventSourcesUpdateCommand(state *appState) *cobra.Command {
	var name string
	var description string
	var configJSON string
	var schemaJSON string
	var signatureHeader string
	var signatureAlgorithm string
	var signatureSecret string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "update <event-source-id>",
		Short: "Update an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.UpdateEventSourceRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			rawSchema := strings.TrimSpace(schemaJSON)
			if rawSchema == "" {
				rawSchema = strings.TrimSpace(configJSON)
			}
			if cmd.Flags().Changed("schema-json") || cmd.Flags().Changed("config-json") {
				if rawSchema != "" && !json.Valid([]byte(rawSchema)) {
					return fmt.Errorf("--schema-json must be valid JSON")
				}
				raw := json.RawMessage(rawSchema)
				req.Schema = &raw
			}
			if cmd.Flags().Changed("signature-header") {
				req.SignatureHeader = &signatureHeader
			}
			if cmd.Flags().Changed("signature-algorithm") {
				req.SignatureAlgorithm = &signatureAlgorithm
			}
			if cmd.Flags().Changed("signature-secret") {
				req.SignatureSecret = &signatureSecret
			}
			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}
			if req.Name == nil && req.Description == nil && req.Schema == nil && req.Enabled == nil &&
				req.SignatureHeader == nil && req.SignatureAlgorithm == nil && req.SignatureSecret == nil {
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
				fmt.Fprintln(os.Stderr, styles.Success("Updated event source "+styles.Bold.Render(styles.SafeText(source.ID))))
				return nil
			}
			return printData(state, source)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "event source name")
	cmd.Flags().StringVar(&description, "description", "", "event source description")
	cmd.Flags().StringVar(&schemaJSON, "schema-json", "", "JSON-encoded event payload schema")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "deprecated alias for --schema-json")
	cmd.Flags().StringVar(&signatureHeader, "signature-header", "", "header containing the event signature")
	cmd.Flags().StringVar(&signatureAlgorithm, "signature-algorithm", "", "signature algorithm (hmac-sha256, stripe-v1, github-sha256)")
	cmd.Flags().StringVar(&signatureSecret, "signature-secret", "", "secret used to verify event signatures")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the source is enabled")

	return cmd
}

func newEventSourcesDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <event-source-id>",
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
				fmt.Fprintln(os.Stderr, styles.Success("Deleted event source "+styles.Bold.Render(styles.SafeText(id))))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": id})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func resolveEventSourceIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrName string) (string, error) {
	if err := validate.SlugOrID(idOrName); err != nil {
		return "", fmt.Errorf("invalid event source identifier: %w", err)
	}
	if validate.IsUUID(idOrName) {
		return idOrName, nil
	}
	_, err := cli.GetEventSource(ctx, idOrName)
	if err == nil {
		return idOrName, nil
	}
	if !client.IsNotFound(err) {
		return "", fmt.Errorf("resolving event source %q: %w", idOrName, err)
	}
	projectID, perr := requireProjectID(state, "")
	if perr != nil {
		return "", fmt.Errorf("project is required to resolve event source %q", idOrName)
	}
	sources, lerr := cli.ListEventSources(ctx, projectID)
	if lerr != nil {
		return "", fmt.Errorf("resolving event source %q: %w", idOrName, lerr)
	}
	for _, s := range sources {
		if s.Name == idOrName {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("event source %q not found", idOrName)
}
