package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newWorkerCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Inspect and drain workers connected to the orchestrator",
		Long: `Worker administration. Workers themselves run on customer infrastructure
using github.com/strait-dev/strait-go/worker — scaffold one with
` + "`" + `strait init --template go-worker` + "`" + ` or
` + "`" + `strait init --template k8s-worker` + "`" + `.`,
	}
	cmd.AddCommand(newWorkerStatusCommand(state))
	cmd.AddCommand(newWorkerDrainCommand(state))
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
