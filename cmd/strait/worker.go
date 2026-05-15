package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

// ErrWorkerSDKRequired is returned by `worker start` and `worker logs`. Those
// subcommands need the long-lived gRPC streaming client that lives in the Go
// SDK (github.com/strait-dev/strait-go) — until that ships the CLI cannot
// run a worker in-process. The orchestration server, status and drain
// subcommands work today.
const workerSDKRequired = "this subcommand requires github.com/strait-dev/strait-go v0.2.0+; pin the SDK in your project and use `strait deploy push` to register jobs"

func newWorkerCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Run and manage Strait workers",
		Long: `Manage long-lived workers that connect to the Strait orchestrator via
gRPC and execute SDK-defined jobs.`,
	}
	cmd.AddCommand(newWorkerStartCommand(state))
	cmd.AddCommand(newWorkerStatusCommand(state))
	cmd.AddCommand(newWorkerDrainCommand(state))
	cmd.AddCommand(newWorkerLogsCommand(state))
	return cmd
}

func newWorkerStartCommand(state *appState) *cobra.Command {
	var queues []string
	var concurrency int
	var name string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run a worker process that pulls tasks from the orchestrator",
		Long: `Starts a long-lived worker that connects to the orchestrator via gRPC
and executes registered jobs. Requires the strait-go SDK to run.

Use Ctrl-C to gracefully drain the worker.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			_ = queues
			_ = concurrency
			_ = name
			_ = state
			return fmt.Errorf("%s", workerSDKRequired)
		},
	}
	cmd.Flags().StringSliceVar(&queues, "queue", []string{"default"}, "queue(s) to pull tasks from")
	cmd.Flags().IntVar(&concurrency, "concurrency", 16, "maximum concurrent tasks")
	cmd.Flags().StringVar(&name, "name", "", "human-readable worker name")
	return cmd
}

func newWorkerStatusCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "List connected workers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			workers, err := cli.ListWorkers(cmd.Context(), state.opts.projectID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				if len(workers) == 0 {
					fmt.Fprintln(os.Stderr, styles.MutedStyle.Render("No workers connected"))
					return nil
				}
				for _, w := range workers {
					fmt.Fprintln(os.Stderr, styles.KeyValue("ID", w.ID))
					fmt.Fprintln(os.Stderr, styles.KeyValue("Status", w.Status))
					fmt.Fprintln(os.Stderr, styles.KeyValue("Queues", strings.Join(w.Queues, ", ")))
					fmt.Fprintln(os.Stderr, styles.KeyValue("Active", fmt.Sprintf("%d", w.ActiveTasks)))
					fmt.Fprintln(os.Stderr, "")
				}
				return nil
			}
			return printData(state, workers)
		},
	}
	return cmd
}

func newWorkerDrainCommand(state *appState) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "drain <worker-id>",
		Short: "Gracefully drain and disconnect a worker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			if id == "" {
				return fmt.Errorf("worker id is required")
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.DisconnectWorker(cmd.Context(), id); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Drain requested for "+styles.Bold.Render(id)))
				return nil
			}
			return printData(state, map[string]any{"worker_id": id, "drained": true})
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "maximum drain wait (informational, server-enforced)")
	return cmd
}

func newWorkerLogsCommand(state *appState) *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs <worker-id>",
		Short: "Stream logs from a worker",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			_ = follow
			_ = state
			return fmt.Errorf("%s", workerSDKRequired)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream new log lines as they arrive")
	return cmd
}
