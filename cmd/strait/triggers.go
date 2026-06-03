package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

var (
	triggersStreamSleep = time.Sleep
	triggersStreamNow   = time.Now
)

func newTriggersCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "triggers",
		Short: "Manage event triggers",
	}

	cmd.AddCommand(newTriggersListCommand(state))
	cmd.AddCommand(newTriggersGetCommand(state))
	cmd.AddCommand(newTriggersSendCommand(state))
	cmd.AddCommand(newTriggersStreamCommand(state))
	cmd.AddCommand(newTriggersPurgeCommand(state))

	return cmd
}

func newTriggersListCommand(state *appState) *cobra.Command {
	var projectID, status string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List event triggers for a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if status != "" {
				validStatuses := map[string]bool{"waiting": true, "received": true, "timed_out": true, "canceled": true}
				if !validStatuses[status] {
					return fmt.Errorf("invalid --status %q, must be one of: waiting, received, timed_out, canceled", status)
				}
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			triggers, err := cli.ListEventTriggers(cmd.Context(), projectID, status)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Event Triggers", len(triggers)))
				for _, t := range triggers {
					fmt.Fprintf(os.Stderr, "  %s  %s  %s\n",
						styles.Bold.Render(t.EventKey),
						styles.StatusBadge(t.Status),
						styles.MutedStyle.Render(t.SourceType),
					)
				}
				return nil
			}
			rows := make([]map[string]any, 0, len(triggers))
			for _, t := range triggers {
				rows = append(rows, map[string]any{
					"id":           t.ID,
					"event_key":    t.EventKey,
					"status":       t.Status,
					"source_type":  t.SourceType,
					"requested_at": t.RequestedAt,
					"expires_at":   t.ExpiresAt,
				})
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (waiting, received, timed_out, canceled)")
	_ = cmd.RegisterFlagCompletionFunc("status", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"waiting", "received", "timed_out", "canceled"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newTriggersGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <event-key>",
		Short: "Get event trigger by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			trigger, err := cli.GetEventTrigger(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.DetailBox("Event Trigger", []string{
					styles.DetailLine("ID", trigger.ID),
					styles.DetailLine("Event Key", styles.Bold.Render(trigger.EventKey)),
					styles.DetailLine("Status", styles.StatusBadge(trigger.Status)),
					styles.DetailLine("Source", trigger.SourceType),
					styles.DetailLine("Timeout", fmt.Sprintf("%ds", trigger.TimeoutSecs)),
					styles.DetailLine("Requested", styles.RelativeTime(trigger.RequestedAt)),
				}))
				return nil
			}
			return printData(state, map[string]any{
				"id":                   trigger.ID,
				"event_key":            trigger.EventKey,
				"project_id":           trigger.ProjectID,
				"source_type":          trigger.SourceType,
				"workflow_run_id":      trigger.WorkflowRunID,
				"workflow_step_run_id": trigger.WorkflowStepRunID,
				"job_run_id":           trigger.JobRunID,
				"status":               trigger.Status,
				"timeout_secs":         trigger.TimeoutSecs,
				"requested_at":         trigger.RequestedAt,
				"received_at":          trigger.ReceivedAt,
				"expires_at":           trigger.ExpiresAt,
				"error":                trigger.Error,
			})
		},
	}

	return cmd
}

func newTriggersSendCommand(state *appState) *cobra.Command {
	var payload, projectID string
	var raw bool

	cmd := &cobra.Command{
		Use:   "send <event-key>",
		Short: "Send an event to a waiting trigger",
		Long: `Send an event keyed by <event-key>.

By default, dispatches through the typed event-trigger endpoint, which resolves
a waiting trigger created via wait-for-event. With --raw, dispatches through the
raw events endpoint, which fans out to subscriptions attached to the event key
(requires --project).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			var payloadMap map[string]any
			if payload != "" {
				if err := json.Unmarshal([]byte(payload), &payloadMap); err != nil {
					return fmt.Errorf("invalid --payload JSON: %w", err)
				}
			}

			if raw {
				pid, err := requireProjectID(state, projectID)
				if err != nil {
					return fmt.Errorf("--project is required with --raw: %w", err)
				}
				if err := cli.SendRawEvent(cmd.Context(), pid, args[0], payloadMap); err != nil {
					return err
				}
				if isTTYRich(state) {
					fmt.Fprintln(os.Stderr, styles.Success("Dispatched raw event "+styles.Bold.Render(args[0])))
					return nil
				}
				return printData(state, map[string]any{
					"event_key": args[0],
					"raw":       true,
					"sent":      true,
				})
			}

			trigger, err := cli.SendEvent(cmd.Context(), args[0], payloadMap)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Sent event to trigger "+styles.Bold.Render(trigger.EventKey)))
				return nil
			}
			return printData(state, map[string]any{
				"id":        trigger.ID,
				"event_key": trigger.EventKey,
				"status":    trigger.Status,
				"sent":      true,
			})
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "JSON payload to send with the event")
	cmd.Flags().BoolVar(&raw, "raw", false, "dispatch through the raw events endpoint instead of the typed trigger")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID (required with --raw)")

	return cmd
}

func newTriggersStreamCommand(state *appState) *cobra.Command {
	var projectID, status string
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Stream new event triggers as they appear (long-poll)",
		Long: `Poll the event-triggers API at --interval and print each new trigger
to stdout as one JSON line. Use Ctrl+C to stop.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			if interval <= 0 {
				return fmt.Errorf("--interval must be > 0")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			seen := make(map[string]bool)
			// Seed with the current page so we only print newly-arrived
			// triggers after stream start.
			initial, err := cli.ListEventTriggers(cmd.Context(), pid, status)
			if err != nil {
				return err
			}
			for _, t := range initial {
				seen[t.ID] = true
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info(fmt.Sprintf("Streaming event triggers for project=%s (interval=%s)", pid, interval)))
			}

			enc := json.NewEncoder(state.out())
			for {
				if cmd.Context().Err() != nil {
					return nil //nolint:nilerr // ctx cancel is a user-initiated graceful exit
				}
				triggers, err := cli.ListEventTriggers(cmd.Context(), pid, status)
				if err != nil {
					return err
				}
				for _, t := range triggers {
					if seen[t.ID] {
						continue
					}
					seen[t.ID] = true
					_ = enc.Encode(map[string]any{
						"id":           t.ID,
						"event_key":    t.EventKey,
						"status":       t.Status,
						"source_type":  t.SourceType,
						"requested_at": t.RequestedAt,
						"observed_at":  triggersStreamNow().UTC(),
					})
				}
				triggersStreamSleep(interval)
			}
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (waiting, received, timed_out, canceled)")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "poll interval")
	_ = cmd.RegisterFlagCompletionFunc("status", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"waiting", "received", "timed_out", "canceled"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newTriggersPurgeCommand(state *appState) *cobra.Command {
	var olderThanDays int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Purge terminal event triggers older than a given age",
		Long:  "Deletes event triggers in terminal state (received, timed_out, canceled) older than --older-than days. Use --dry-run to preview.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if olderThanDays < 1 {
				return fmt.Errorf("--older-than must be >= 1 day")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			count, err := cli.PurgeEventTriggers(cmd.Context(), olderThanDays, dryRun)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				if dryRun {
					fmt.Fprintln(os.Stderr, styles.Info(fmt.Sprintf("Would delete %d trigger(s)", count)))
				} else {
					fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Purged %d trigger(s)", count)))
				}
				return nil
			}
			if dryRun {
				return printData(state, map[string]any{"dry_run": true, "would_delete": count})
			}
			return printData(state, map[string]any{"deleted": count})
		},
	}

	cmd.Flags().IntVar(&olderThanDays, "older-than", 30, "delete triggers older than N days")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without deleting")

	return cmd
}
