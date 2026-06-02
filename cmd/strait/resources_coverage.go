package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

// registerJobsCoverageCommands adds job sub-commands not present in jobs.go:
// pause, resume, batch-enable, batch-disable, trigger-bulk, version-get,
// remove-dependency.
func registerJobsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newJobsPauseCommand(state))
	parent.AddCommand(newJobsResumeCommand(state))
	parent.AddCommand(newJobsBatchEnableCommand(state))
	parent.AddCommand(newJobsBatchDisableCommand(state))
	parent.AddCommand(newJobsTriggerBulkCommand(state))
	parent.AddCommand(newJobsVersionGetCommand(state))
	parent.AddCommand(newJobsRemoveDependencyCommand(state))
}

func newJobsPauseCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <job-id>",
		Short: "Pause a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.PauseJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Paused job "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, out)
		},
	}
}

func newJobsResumeCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <job-id>",
		Short: "Resume a paused job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ResumeJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Resumed job "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, out)
		},
	}
}

func newJobsBatchEnableCommand(state *appState) *cobra.Command {
	var ids []string
	cmd := &cobra.Command{
		Use:   "batch-enable",
		Short: "Enable multiple jobs by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(ids) == 0 {
				return fmt.Errorf("at least one --id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BatchEnableJobs(cmd.Context(), ids)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&ids, "id", nil, "job ID to enable (repeatable)")
	return cmd
}

func newJobsBatchDisableCommand(state *appState) *cobra.Command {
	var ids []string
	cmd := &cobra.Command{
		Use:   "batch-disable",
		Short: "Disable multiple jobs by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(ids) == 0 {
				return fmt.Errorf("at least one --id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BatchDisableJobs(cmd.Context(), ids)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringArrayVar(&ids, "id", nil, "job ID to disable (repeatable)")
	return cmd
}

func newJobsTriggerBulkCommand(state *appState) *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "trigger-bulk <job-id>",
		Short: "Bulk-trigger multiple runs for a job from a JSON file",
		Long:  `Bulk-trigger runs for a job. Provide the body ({"items":[...]}) via --from-file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job id: %w", err)
			}
			body, err := readJSONFileBody(fromFile)
			if err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.BulkTriggerJobRaw(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file containing the bulk trigger body ({\"items\":[...]})")
	return cmd
}

func newJobsVersionGetCommand(state *appState) *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "version-get <job-id>",
		Short: "Get a specific version of a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job id: %w", err)
			}
			if strings.TrimSpace(versionID) == "" {
				return fmt.Errorf("--version is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetJobVersion(cmd.Context(), args[0], versionID)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&versionID, "version", "", "version ID")
	return cmd
}

func newJobsRemoveDependencyCommand(state *appState) *cobra.Command {
	var depID string
	cmd := &cobra.Command{
		Use:   "remove-dependency <job-id>",
		Short: "Remove a dependency from a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid job id: %w", err)
			}
			if strings.TrimSpace(depID) == "" {
				return fmt.Errorf("--dep-id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteJobDependency(cmd.Context(), args[0], depID); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Removed dependency "+styles.Bold.Render(depID)+" from job "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "dep_id": depID})
		},
	}
	cmd.Flags().StringVar(&depID, "dep-id", "", "dependency ID to remove")
	return cmd
}

// registerEventSourcesCoverageCommands adds event-source sub-commands not
// present in event_sources.go: subscriptions, subscribe, unsubscribe.
func registerEventSourcesCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newEventSourcesSubscriptionsCommand(state))
	parent.AddCommand(newEventSourcesSubscribeCommand(state))
	parent.AddCommand(newEventSourcesUnsubscribeCommand(state))
}

func newEventSourcesSubscriptionsCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "subscriptions <source-id>",
		Short: "List subscriptions for an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid event source id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListEventSourceSubscriptions(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

func newEventSourcesSubscribeCommand(state *appState) *cobra.Command {
	var targetType string
	var targetID string
	var enabled bool
	var enabledSet bool
	var filterExpr string

	cmd := &cobra.Command{
		Use:   "subscribe <source-id>",
		Short: "Subscribe a target to an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid event source id: %w", err)
			}
			if strings.TrimSpace(targetType) == "" {
				return fmt.Errorf("--target-type is required")
			}
			if strings.TrimSpace(targetID) == "" {
				return fmt.Errorf("--target-id is required")
			}
			var enabledPtr *bool
			if enabledSet {
				v := enabled
				enabledPtr = &v
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.SubscribeEventSource(cmd.Context(), args[0], targetType, targetID, enabledPtr, filterExpr)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "subscription target type (required)")
	cmd.Flags().StringVar(&targetID, "target-id", "", "subscription target ID (required)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the subscription is enabled")
	cmd.Flags().StringVar(&filterExpr, "filter", "", "filter expression")

	// Track whether --enabled was explicitly provided.
	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		enabledSet = cmd.Flags().Changed("enabled")
		return nil
	}

	return cmd
}

func newEventSourcesUnsubscribeCommand(state *appState) *cobra.Command {
	var subID string
	cmd := &cobra.Command{
		Use:   "unsubscribe <source-id>",
		Short: "Remove a subscription from an event source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid event source id: %w", err)
			}
			if strings.TrimSpace(subID) == "" {
				return fmt.Errorf("--sub-id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DeleteEventSourceSubscription(cmd.Context(), args[0], subID); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Removed subscription "+styles.Bold.Render(subID)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "sub_id": subID})
		},
	}
	cmd.Flags().StringVar(&subID, "sub-id", "", "subscription ID to remove")
	return cmd
}

// registerSecretsCoverageCommands adds secrets sub-commands not present in the
// existing secrets.go: get.
func registerSecretsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newSecretsCoverageGetCommand(state))
}

func newSecretsCoverageGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "get <secret-id>",
		Short: "Get a secret by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid secret id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetServerSecret(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

// registerNotificationsCoverageCommands adds notifications sub-commands not
// present in the existing notifications.go: deliveries.
func registerNotificationsCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newNotificationsDeliveriesCommand(state))
}

func newNotificationsDeliveriesCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "deliveries",
		Short: "List notification delivery records",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListNotificationDeliveries(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}

// registerAPIKeysCoverageCommands adds api-keys sub-commands not present in
// the existing api_keys.go: expiring-soon.
func registerAPIKeysCoverageCommands(parent *cobra.Command, state *appState) {
	parent.AddCommand(newAPIKeysExpiringSoonCommand(state))
}

func newAPIKeysExpiringSoonCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "expiring-soon",
		Short: "List API keys that are expiring soon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListExpiringAPIKeys(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
}
