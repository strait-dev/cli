package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newWorkflowRunsPauseCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <workflow-run-id>",
		Short: "Pause an in-flight workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.PauseWorkflowRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Paused workflow run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, run)
		},
	}
}

func newWorkflowRunsResumeCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <workflow-run-id>",
		Short: "Resume a paused workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.ResumeWorkflowRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Resumed workflow run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, run)
		},
	}
}

func newWorkflowRunsRetryCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <workflow-run-id>",
		Short: "Retry a failed workflow run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			run, err := cli.RetryWorkflowRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Retried workflow run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, run)
		},
	}
}

func newWorkflowRunsApproveStepCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "approve <workflow-run-id> <step-ref>",
		Short: "Approve a workflow step pending review",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(args[1]); err != nil {
				return fmt.Errorf("invalid step ref: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.ApproveWorkflowStep(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Approved step "+styles.Bold.Render(args[1])+" on run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]string{"run_id": args[0], "step_ref": args[1], "status": "approved"})
		},
	}
}

func newWorkflowRunsRetryStepCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <workflow-run-id> <step-ref>",
		Short: "Retry an individual workflow step",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(args[1]); err != nil {
				return fmt.Errorf("invalid step ref: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.RetryWorkflowStep(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Retried step "+styles.Bold.Render(args[1])+" on run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]string{"run_id": args[0], "step_ref": args[1], "status": "retried"})
		},
	}
}

func newWorkflowRunsSkipStepCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "skip <workflow-run-id> <step-ref>",
		Short: "Skip a workflow step",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(args[1]); err != nil {
				return fmt.Errorf("invalid step ref: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.SkipWorkflowStep(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Skipped step "+styles.Bold.Render(args[1])+" on run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]string{"run_id": args[0], "step_ref": args[1], "status": "skipped"})
		},
	}
}

func newWorkflowRunsForceCompleteStepCommand(state *appState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "force-complete <workflow-run-id> <step-ref>",
		Short: "Force-complete a workflow step regardless of state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid workflow run id: %w", err)
			}
			if err := validate.SlugOrID(args[1]); err != nil {
				return fmt.Errorf("invalid step ref: %w", err)
			}
			if err := requireConfirmation(state, fmt.Sprintf("Force-complete step %s on run %s?", args[1], args[0]), yes); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			if err := cli.ForceCompleteWorkflowStep(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Force-completed step "+styles.Bold.Render(args[1])+" on run "+styles.Bold.Render(args[0])))
				return nil
			}
			return printData(state, map[string]string{"run_id": args[0], "step_ref": args[1], "status": "force-completed"})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}

func newWorkflowRunsApproveStepAliasCommand(state *appState) *cobra.Command {
	cmd := newWorkflowRunsApproveStepCommand(state)
	cmd.Use = "approve-step <workflow-run-id> <step-ref>"
	return cmd
}

func newWorkflowRunsRetryStepAliasCommand(state *appState) *cobra.Command {
	cmd := newWorkflowRunsRetryStepCommand(state)
	cmd.Use = "retry-step <workflow-run-id> <step-ref>"
	return cmd
}

func newWorkflowRunsSkipStepAliasCommand(state *appState) *cobra.Command {
	cmd := newWorkflowRunsSkipStepCommand(state)
	cmd.Use = "skip-step <workflow-run-id> <step-ref>"
	return cmd
}

func newWorkflowRunsForceCompleteStepAliasCommand(state *appState) *cobra.Command {
	cmd := newWorkflowRunsForceCompleteStepCommand(state)
	cmd.Use = "force-complete-step <workflow-run-id> <step-ref>"
	return cmd
}
