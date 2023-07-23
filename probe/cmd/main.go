package main

import (
	"context"
	"log"

	"github.com/isac322/buildkit-state/probe/internal"
	"github.com/isac322/buildkit-state/probe/internal/buildkit"

	"github.com/docker/docker/client"
	"github.com/sethvargo/go-githubactions"
	"github.com/spf13/cobra"
	_ "go.uber.org/automaxprocs"
)

var (
	rootCmd = &cobra.Command{
		Use:   "buildkit-state",
		Short: "Manage buildkit state intrusively",
	}
	loadCmd = &cobra.Command{
		Use:   "load",
		Args:  cobra.NoArgs,
		Short: "Download buildkit state from remote and inject into docker",
		RunE:  load,
	}
	saveCmd = &cobra.Command{
		Use:   "save",
		Args:  cobra.NoArgs,
		Short: "Extract and update buildkit state to remote",
		RunE:  save,
	}
)

func init() {
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(saveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func load(cmd *cobra.Command, _ []string) error {
	return run(cmd.Context(), false)
}

func save(cmd *cobra.Command, _ []string) error {
	return run(cmd.Context(), true)
}

func run(ctx context.Context, isSave bool) error {
	gha := githubactions.New()
	manager, err := newManager(ctx, gha)
	if err != nil {
		return err
	}

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		gha.Errorf("Failed connect docker: %+v", err)
		return err
	}

	builderName := gha.GetInput(inputBuildxName)
	bkCli, err := buildkit.NewContainerizedDriver(ctx, docker, builderName)
	if err != nil {
		return err
	}

	if isSave {
		err = internal.SaveFromContainerToRemote(ctx, gha, bkCli, manager)
		if err != nil {
			gha.Errorf("Failed to save buildkit state: %+v", err)
			return err
		}
	} else {
		err = internal.LoadFromRemoteToContainer(ctx, gha, bkCli, manager)
		if err != nil {
			gha.Errorf("Failed to restore buildkit state: %+v", err)
			return err
		}
	}

	return nil
}
