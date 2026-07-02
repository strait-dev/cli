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

	deleteCmd := newWebhooksDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeWebhookIDs(state)
	getCmd := newWebhooksGetCommand(state)
	getCmd.ValidArgsFunction = completeWebhookIDs(state)
	rotateCmd := newWebhooksRotateSecretCommand(state)
	rotateCmd.ValidArgsFunction = completeWebhookIDs(state)

	cmd.AddCommand(newWebhooksListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newWebhooksCreateCommand(state))
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(rotateCmd)
	cmd.AddCommand(newWebhooksDeliveriesCommand(state))
	cmd.AddCommand(newWebhooksRetryCommand(state))
	cmd.AddCommand(newWebhooksTestCommand(state))

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
	var projectID string
	var reveal bool
	cmd := &cobra.Command{
		Use:   "get <webhook-id>",
		Short: "Get webhook subscription details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			hook, err := cli.GetWebhook(cmd.Context(), projectID, args[0])
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
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
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
			if cmd.Flags().Changed("secret") {
				return fmt.Errorf("--secret is no longer supported; the server generates and returns a signing secret on create")
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

func newWebhooksRotateSecretCommand(state *appState) *cobra.Command {
	var gracePeriod int
	cmd := &cobra.Command{
		Use:   "rotate-secret <webhook-id>",
		Short: "Rotate a webhook subscription signing secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid webhook id: %w", err)
			}
			if gracePeriod < 0 {
				return fmt.Errorf("--grace-period-minutes must be >= 0")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			resp, err := cli.RotateWebhookSecret(cmd.Context(), args[0], client.RotateWebhookSecretRequest{
				GracePeriodMinutes: gracePeriod,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Rotated webhook secret "+styles.Bold.Render(styles.SafeText(resp.SubscriptionID))))
				fmt.Fprintln(os.Stderr, styles.MutedStyle.Render("(save the new secret now -- it will not be shown again)"))
				fmt.Fprintln(os.Stderr, "secret: "+resp.NewSecret)
				return nil
			}
			return printData(state, resp)
		},
	}
	cmd.Flags().IntVar(&gracePeriod, "grace-period-minutes", 60, "minutes to accept the previous secret")
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
	var status string
	var cursor string
	cmd := &cobra.Command{
		Use:   "deliveries",
		Short: "List webhook delivery attempts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			deliveries, err := cli.ListWebhookDeliveries(cmd.Context(), status, limit, cursor)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(deliveries))
			for _, d := range deliveries {
				rows = append(rows, map[string]any{
					"id":               d.ID,
					"subscription_id":  d.SubscriptionID,
					"status":           d.Status,
					"attempts":         d.Attempts,
					"max_attempts":     d.MaxAttempts,
					"last_status_code": d.LastStatusCode,
					"created_at":       d.CreatedAt,
				})
			}
			return printData(state, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max deliveries to return")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (pending, delivered, failed, dead)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor")
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
	var hookURL string
	var secret string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a synthetic test event to a webhook URL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(hookURL) == "" {
				return fmt.Errorf("--url is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			resp, err := cli.TestWebhook(cmd.Context(), client.TestWebhookRequest{
				URL:    hookURL,
				Secret: secret,
			})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				if resp.Success {
					fmt.Fprintln(os.Stderr, styles.Success("Webhook test succeeded"))
				} else {
					fmt.Fprintln(os.Stderr, styles.Warn("Webhook test failed"))
				}
				return nil
			}
			return printData(state, resp)
		},
	}
	cmd.Flags().StringVar(&hookURL, "url", "", "webhook target URL")
	cmd.Flags().StringVar(&secret, "secret", "", "optional signing secret")
	return cmd
}
