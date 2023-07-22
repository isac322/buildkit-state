package internal

import (
	"context"
	"io"
	"strconv"

	"github.com/isac322/buildkit-state/probe/internal/buildkit"
	gha2 "github.com/isac322/buildkit-state/probe/internal/gha"

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
	bkCli buildkit.Driver,
	loader Loader,
) (err error) {
	var loaded LoadedCache
	var found bool

	func() {
		gha.Group("Load cache from remote")
		defer gha.EndGroup()

		primaryKey := gha.GetInput(inputPrimaryKey)
		gha.Debugf("primary key: %v", primaryKey)
		secondaryKeys := gha2.GetMultilineInput(gha, inputSecondaryKeys)
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

	func() {
		gha.Group("Load cache to docker")
		defer gha.EndGroup()

		gha.Infof("stopping buildkitd...")
		err = bkCli.Stop(ctx)
		if err != nil {
			gha.Errorf("Failed to stop buildkitd container: %+v", err)
			return
		}

		gha.Infof("restoring cache into buildkitd...")
		err = DecompressZstdTo(ctx, bkCli, loaded.Data)
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
		err = bkCli.Resume(ctx)
		if err != nil {
			gha.Errorf("Failed to resume buildkitd container: %+v", err)
			return
		}

		var usage []byte
		usage, err = bkCli.PrintDiskUsage(ctx)
		if err != nil {
			gha.Errorf("Failed to print disk usage: %+v", err)
			return
		}
		gha.Infof(string(usage))
	}()

	return err
}
