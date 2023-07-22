package main

import (
	"github.com/isac322/buildkit-state/probe/internal"

	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load",
	Args:  cobra.NoArgs,
	Short: "Download buildkit state from remote and inject into docker",
	RunE:  load,
}

func init() {
	rootCmd.AddCommand(loadCmd)
}

func load(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	gha, bkCli, loader, err := newDependencies(ctx)
	if err != nil {
		return err
	}

	err = internal.LoadFromRemoteToContainer(ctx, gha, bkCli, loader)
	if err != nil {
		gha.Errorf("Failed to restore buildkit state: %+v", err)
		return err
	}

	return nil
}
