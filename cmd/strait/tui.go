package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/strait-dev/cli/internal/tui/dashboard"
)

// newTUICommand returns `strait tui`, the interactive dashboard. The dashboard
// is the single GUI surface for the CLI: a k9s-style pane switcher exposing
// the most common read paths (jobs, runs, workflows, workflow runs). All write
// actions remain in the regular CLI surface — keeping the TUI read-only bounds
// the maintenance cost.
func newTUICommand(state *appState) *cobra.Command {
	var projectID string
	var refresh time.Duration

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive dashboard (jobs, runs, workflows)",
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

			model, err := dashboard.New(dashboard.Config{
				Loader:       cli,
				ProjectID:    projectID,
				RefreshEvery: refresh,
			})
			if err != nil {
				return err
			}

			program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(cmd.Context()))
			if _, err := program.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().DurationVar(&refresh, "refresh", 5*time.Second, "auto-refresh interval")

	return cmd
}
