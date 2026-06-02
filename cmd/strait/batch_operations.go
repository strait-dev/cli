package main

import (
	"fmt"

	"github.com/strait-dev/cli/internal/validate"

	"github.com/spf13/cobra"
)

func newBatchOperationsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-operations",
		Short: "Manage batch operations",
	}
	cmd.AddCommand(newBatchOperationsListCommand(state))
	cmd.AddCommand(newBatchOperationsGetCommand(state))
	return cmd
}

func newBatchOperationsListCommand(state *appState) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List batch operations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.ListBatchOperations(cmd.Context(), limit)
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	return cmd
}

func newBatchOperationsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <batch-id>",
		Short: "Get a batch operation by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.SlugOrID(args[0]); err != nil {
				return fmt.Errorf("invalid batch operation id: %w", err)
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			out, err := cli.GetBatchOperation(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printData(state, out)
		},
	}
	return cmd
}
