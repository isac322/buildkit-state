package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/isac322/buildkit-state/probe/internal"
	"github.com/isac322/buildkit-state/probe/internal/buildkit"
	localmanager "github.com/isac322/buildkit-state/probe/internal/remote/local"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:       "buildkit-state",
		ValidArgs: []string{"save", "load"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short:     "Manage buildkit state intrusively",
		RunE:      run,
	}

	builderName      string
	destinationPath  string
	primaryKey       string
	secondaryKeys    []string
	targetTypes      []string
	compressionLevel int
)

func init() {
	rootCmd.Flags().StringVarP(&builderName, "builder", "b", "", "builx builder name")
	panicIf(rootCmd.MarkFlagRequired("builder"))
	rootCmd.Flags().StringVarP(&destinationPath, "dest", "d", ".", "cache directory")
	rootCmd.Flags().StringVarP(&primaryKey, "cache-key", "c", "", "cache key")
	panicIf(rootCmd.MarkFlagRequired("cache-key"))
	rootCmd.Flags().StringSliceVarP(&secondaryKeys, "cache-restore-key", "r", nil, "cache restore key")
	rootCmd.Flags().StringSliceVarP(
		&targetTypes,
		"target-types",
		"t",
		[]string{"exec.cachemount", "front"},
		"buildkit state types",
	)
	rootCmd.Flags().IntVarP(&compressionLevel, "compression", "l", 19, "zstd compression level")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	gha := githubactions.New(githubactions.WithGetenv(func(key string) string {
		switch key {
		case "INPUT_RESUME-BUILDER":
			return "false"
		case "INPUT_CACHE-KEY":
			return primaryKey
		case "INPUT_CACHE-RESTORE-KEYS":
			return strings.Join(secondaryKeys, "\n")
		case "INPUT_TARGET-TYPES":
			return strings.Join(targetTypes, "\n")
		case "INPUT_COMPRESSION-LEVEL":
			return strconv.Itoa(compressionLevel)
		case "GITHUB_OUTPUT":
			return "/dev/null"
		case "GITHUB_STATE":
			return "/dev/null"
		default:
			return ""
		}
	}))
	manager := localmanager.New(destinationPath)
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		gha.Errorf("Failed connect docker: %+v", err)
		return err
	}

	info, err := docker.Info(ctx)
	if err != nil {
		gha.Errorf("Failed info docker: %+v", err)
		return err
	}
	gha.Infof("%+v", info)

	bkCli, err := buildkit.NewContainerizedDriver(ctx, docker, builderName)
	if err != nil {
		gha.Errorf("Failed to connect buildkit: %+v", err)
		return err
	}

	switch args[0] {
	case "load":
		return internal.LoadFromRemoteToContainer(ctx, gha, bkCli, manager)
	case "save":
		return internal.SaveFromContainerToRemote(ctx, gha, bkCli, manager)
	default:
		return errors.Errorf("unknown command: %+v", args[0])
	}
}

func panicIf(err error) {
	if err == nil {
		return
	}

	log.Panicf("%+v", err)
}
