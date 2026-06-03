package main

import (
	"github.com/spf13/cobra"
)

func newStatsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show account and project statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetStats(cmd.Context())
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	return cmd
}
