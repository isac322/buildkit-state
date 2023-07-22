package main

import (
	"github.com/isac322/buildkit-state/probe/internal"

	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Args:  cobra.NoArgs,
	Short: "Extract and update buildkit state to remote",
	RunE:  save,
}

func init() {
	rootCmd.AddCommand(saveCmd)
}

func save(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	gha, bkCli, saver, err := newDependencies(ctx)
	if err != nil {
		return err
	}

	err = internal.SaveFromContainerToRemote(ctx, gha, bkCli, saver)
	if err != nil {
		gha.Errorf("Failed to restore buildkit state: %+v", err)
		return err
	}

	return nil
}
