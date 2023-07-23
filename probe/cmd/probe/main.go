package main

import (
	"context"
	"log"

	"github.com/isac322/buildkit-state/probe/internal"
	"github.com/isac322/buildkit-state/probe/internal/buildkit"
	"github.com/isac322/buildkit-state/probe/internal/remote"

	"github.com/docker/docker/client"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
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

	dockerEndpoint string
)

func init() {
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(saveCmd)

	rootCmd.PersistentFlags().StringVarP(
		&dockerEndpoint,
		"docker-endpoint",
		"e",
		client.DefaultDockerHost,
		"Endpoint of docker daemon",
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func load(cmd *cobra.Command, _ []string) error {
	return run(cmd.Context(), internal.LoadFromRemoteToContainer)
}

func save(cmd *cobra.Command, _ []string) error {
	return run(cmd.Context(), internal.SaveFromContainerToRemote)
}

func run(ctx context.Context, worker Worker) error {
	gha := githubactions.New()
	manager, err := newManager(ctx, gha)
	if err != nil {
		return err
	}

	gha.Infof("Connecting to docker...")
	docker, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHost(dockerEndpoint),
	)
	if err != nil {
		gha.Errorf("Failed connect docker: %+v", err)
		return err
	}

	if gha.Getenv("RUNNER_DEBUG") == "1" {
		gha.Infof("Printing docker info...")
		info, err := docker.Info(ctx)
		if err != nil {
			gha.Errorf("Failed info docker: %+v", err)
			return err
		}
		escape, err := json.MarshalIndentWithOption(info, "", "    ", json.DisableHTMLEscape())
		if err != nil {
			err = errors.WithStack(err)
			gha.Errorf("Failed marshal docker info into json: %+v", err)
			return err
		}
		gha.Infof("%+s", string(escape))
	}

	gha.Infof("Connecting to buildkit daemon...")
	builderName := gha.GetInput(inputBuildxName)
	bkCli, err := buildkit.NewContainerizedDriver(ctx, docker, builderName)
	if err != nil {
		gha.Errorf("Failed to connect buildkit: %+v", err)
		return err
	}

	return worker(ctx, gha, bkCli, manager)
}

type Worker func(context.Context, *githubactions.Action, buildkit.Driver, remote.Manager) error
