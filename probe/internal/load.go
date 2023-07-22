package internal

import (
	"context"
	"io"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	bkclient "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/samber/mo"
	"github.com/sethvargo/go-githubactions"
)

type LoadedCache struct {
	Key   string
	Data  io.ReadCloser
	Extra map[string]any
}

type Loader interface {
	Load(ctx context.Context, primaryKey string, secondaryKeys []string) (mo.Option[LoadedCache], error)
}

func LoadFromRemoteToContainer(
	ctx context.Context,
	gha *githubactions.Action,
	docker client.CommonAPIClient,
	buildkit *bkclient.Client,
	builderName string,
	loader Loader,
) (err error) {
	var loaded LoadedCache
	var found bool

	func() {
		gha.Group("Load cache from remote")
		defer gha.EndGroup()

		primaryKey := gha.GetInput(inputPrimaryKey)
		gha.Debugf("primary key: %v", primaryKey)
		secondaryKeys := getMultilineInput(gha, inputSecondaryKeys)
		gha.Debugf("secondary keys: %v", secondaryKeys)

		var cache mo.Option[LoadedCache]
		cache, err = loader.Load(ctx, primaryKey, secondaryKeys)
		if err != nil {
			gha.Errorf("Failed to load cache from remote: %+v", err)
			return
		}

		loaded, found = cache.Get()
		if !found {
			gha.Infof("Can not find cache.\nskip state loading.")
			return
		}
		gha.Infof("found cache from key: %v", loaded.Key)
		gha.SetOutput(outputRestoredCacheKey, loaded.Key)
	}()
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	gha.SaveState(stateLoadedCacheKey, loaded.Key)
	containerName := BuildKitContainerNameFromBuilder(builderName)

	func() {
		gha.Group("Load cache to docker")
		defer gha.EndGroup()

		gha.Infof("stopping buildkitd...")
		err = docker.ContainerStop(ctx, containerName, container.StopOptions{})
		if err != nil {
			gha.Errorf("Failed to stop buildkitd container: %+v", err)
			return
		}

		gha.Infof("restoring cache into buildkitd...")
		err = DecompressZstdTo(ctx, docker, containerName, loaded.Data)
		if err != nil {
			gha.Errorf("Failed to restore cache into buildkitd: %+v", err)
			return
		}
	}()
	if err != nil {
		return err
	}

	resumeBuildkitD, err := strconv.ParseBool(gha.GetInput(inputResumeBuilder))
	if err != nil {
		gha.Errorf(`Failed to parse "%s": %+v`, inputResumeBuilder, err)
		return errors.WithStack(err)
	}
	if !resumeBuildkitD {
		gha.Debugf("Skip resuming")
		return nil
	}

	func() {
		gha.Group("Resume buildkitd")
		defer gha.EndGroup()

		gha.Infof("starting buildkitd...")
		err = docker.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
		if err != nil {
			gha.Errorf("Failed to resume buildkitd container: %+v", err)
			return
		}

		err = printDiskUsage(ctx, gha, buildkit)
	}()

	return err
}
