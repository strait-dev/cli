package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

const webhookSecretMask = "********"

func newWebhooksCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Manage outbound webhook subscriptions",
	}

	getCmd := newWebhooksGetCommand(state)
	getCmd.ValidArgsFunction = completeWebhookIDs(state)
	updateCmd := newWebhooksUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeWebhookIDs(state)
	deleteCmd := newWebhooksDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeWebhookIDs(state)
	deliveriesCmd := newWebhooksDeliveriesCommand(state)
	deliveriesCmd.ValidArgsFunction = completeWebhookIDs(state)
	testCmd := newWebhooksTestCommand(state)
	testCmd.ValidArgsFunction = completeWebhookIDs(state)

	cmd.AddCommand(newWebhooksListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newWebhooksCreateCommand(state))
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(deliveriesCmd)
	cmd.AddCommand(newWebhooksRetryCommand(state))
	cmd.AddCommand(testCmd)

	return cmd
}

func maskWebhookSecret(secret string, reveal bool) string {
	if reveal || secret == "" {
		return secret
	}
	return webhookSecretMask
}

func newWebhooksListCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhook subscriptions",
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
			hooks, err := cli.ListWebhooks(cmd.Context(), projectID)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(hooks))
			for _, h := range hooks {
				rows = append(rows, map[string]any{
					"id":     h.ID,
					"url":    h.URL,
					"events": strings.Join(h.Events, ","),
					"active": h.Active,
				})
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Webhooks", len(hooks)))
				for _, h := range hooks {
					fmt.Fprintf(os.Stderr, "  %s  %s  events=%s  %s\n",
						styles.Enabled(h.Active),
						styles.SafeText(h.URL),
						styles.MutedStyle.Render(styles.SafeText(strings.Join(h.Events, ","))),
						styles.MutedStyle.Render(styles.SafeText(h.ID)),
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

func newWebhooksGetCommand(state *appState) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "get <webhook-id>",
		Short: "Get webhook subscription details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			hook, err := cli.GetWebhook(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			masked := *hook
			masked.Secret = maskWebhookSecret(hook.Secret, reveal)
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", masked.ID),
					styles.DetailLine("URL", masked.URL),
					styles.DetailLine("Events", strings.Join(masked.Events, ",")),
					styles.DetailLine("Active", styles.Enabled(masked.Active)),
					styles.DetailLine("Secret", masked.Secret),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Webhook", lines))
				return nil
			}
			return printData(state, &masked)
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "show webhook secret in plaintext")
	return cmd
}

func newWebhooksCreateCommand(state *appState) *cobra.Command {
	var projectID string
	var hookURL string
	var events []string
	var secret string
	var active bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook subscription",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(hookURL) == "" {
				return fmt.Errorf("--url is required")
			}
			if len(events) == 0 {
				return fmt.Errorf("--event is required (repeatable)")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			req := client.CreateWebhookRequest{
				ProjectID: projectID,
				URL:       hookURL,
				Events:    events,
				Secret:    secret,
			}
			if cmd.Flags().Changed("active") {
				req.Active = &active
			}
			hook, err := cli.CreateWebhook(cmd.Context(), req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created webhook "+styles.Bold.Render(styles.SafeText(hook.ID))))
				if hook.Secret != "" {
					fmt.Fprintln(os.Stderr, styles.MutedStyle.Render("(save the secret now -- it will not be shown again unless --reveal)"))
					fmt.Fprintln(os.Stderr, "secret: "+hook.Secret)
				}
				return nil
			}
			return printData(state, hook)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&hookURL, "url", "", "webhook target URL")
	cmd.Flags().StringArrayVar(&events, "event", nil, "event type to subscribe to (repeatable)")
	cmd.Flags().StringVar(&secret, "secret", "", "shared secret used for HMAC signing")
	cmd.Flags().BoolVar(&active, "active", true, "whether the webhook is active")

	return cmd
}

func newWebhooksUpdateCommand(state *appState) *cobra.Command {
	var hookURL string
	var events []string
	var secret string
	var active bool

	cmd := &cobra.Command{
		Use:   "update <webhook-id>",
		Short: "Update a webhook subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			req := client.UpdateWebhookRequest{}
			if cmd.Flags().Changed("url") {
				req.URL = &hookURL
			}
			if cmd.Flags().Changed("event") {
				req.Events = &events
			}
			if cmd.Flags().Changed("secret") {
				req.Secret = &secret
			}
			if cmd.Flags().Changed("active") {
				req.Active = &active
			}
			if req.URL == nil && req.Events == nil && req.Secret == nil && req.Active == nil {
				return fmt.Errorf("at least one update flag is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			hook, err := cli.UpdateWebhook(cmd.Context(), args[0], req)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated webhook "+styles.Bold.Render(styles.SafeText(hook.ID))))
				return nil
			}
			masked := *hook
			masked.Secret = maskWebhookSecret(hook.Secret, false)
			return printData(state, &masked)
		},
	}

	cmd.Flags().StringVar(&hookURL, "url", "", "webhook target URL")
	cmd.Flags().StringArrayVar(&events, "event", nil, "event types (replaces existing list)")
	cmd.Flags().StringVar(&secret, "secret", "", "shared secret used for HMAC signing")
	cmd.Flags().BoolVar(&active, "active", true, "whether the webhook is active")

	return cmd
}

func newWebhooksDeleteCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <webhook-id>",
		Short: "Delete a webhook subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			if err := requireConfirmation(state, "Delete this webhook?", yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteWebhook(cmd.Context(), args[0]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted webhook "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": args[0]})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func newWebhooksDeliveriesCommand(state *appState) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "deliveries <webhook-id>",
		Short: "List webhook delivery attempts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			deliveries, err := cli.ListWebhookDeliveries(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(deliveries))
			for _, d := range deliveries {
				rows = append(rows, map[string]any{
					"id":            d.ID,
					"event_type":    d.EventType,
					"status":        d.Status,
					"status_code":   d.StatusCode,
					"attempt_count": d.AttemptCount,
					"requested_at":  d.RequestedAt,
				})
			}
			return printData(state, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max deliveries to return")
	return cmd
}

func newWebhooksRetryCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <delivery-id>",
		Short: "Retry a failed webhook delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid delivery id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			delivery, err := cli.RetryWebhookDelivery(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Retry queued: "+styles.Bold.Render(styles.SafeText(delivery.ID))))
				return nil
			}
			return printData(state, delivery)
		},
	}
	return cmd
}

func newWebhooksTestCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <webhook-id>",
		Short: "Send a synthetic test event to a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			resp, err := cli.TestWebhook(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Test dispatched: "+styles.Bold.Render(styles.SafeText(resp.DeliveryID))))
				return nil
			}
			return printData(state, resp)
		},
	}
	return cmd
}
