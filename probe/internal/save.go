package internal

import (
	"bytes"
	"context"
	"strconv"

	"github.com/isac322/buildkit-state/probe/internal/buildkit"
	gha2 "github.com/isac322/buildkit-state/probe/internal/gha"
	"github.com/isac322/buildkit-state/probe/internal/remote"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
)

func SaveFromContainerToRemote(
	ctx context.Context,
	gha *githubactions.Action,
	bkCli buildkit.Driver,
	manager remote.Manager,
) (err error) {
	defer func() {
		if err != nil {
			gha.Errorf("Failed to load/save buildkit state: %+v", err)
		}
	}()

	if gha.Getenv("RUNNER_DEBUG") == "1" {
		usage, err := bkCli.PrintDiskUsage(ctx)
		if err != nil {
			gha.Errorf("Failed to print disk usage: %+v", err)
			return err
		}

		gha.Debugf(string(usage))
	}

	cacheKey := gha.GetInput(inputPrimaryKey)
	restoredCacheKey := gha.Getenv("STATE_" + stateLoadedCacheKey)
	gha.Infof("restoredCacheKey: %s, cacheKey: %s", restoredCacheKey, cacheKey)

	if cacheKey == restoredCacheKey {
		rewriteCache, err := strconv.ParseBool(gha.GetInput(inputRewriteCache))
		if err != nil {
			gha.Errorf(`Failed to parse "%s": %+v`, inputRewriteCache, err)
			return errors.WithStack(err)
		}
		if !rewriteCache {
			gha.Infof("Cache key matched. Ignore cache saving.")
			return nil
		}
	}

	func() {
		gha.Group("Remove unwanted caches")
		defer gha.EndGroup()

		targetTypes := gha2.GetMultilineInput(gha, inputTargetTypes)
		err = bkCli.PruneExcept(ctx, targetTypes)
		if err != nil {
			gha.Errorf(`Failed to prune caches: %+v`, err)
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
	if err != nil {
		return err
	}

	func() {
		gha.Group("Save buildkit state to remote")
		defer gha.EndGroup()

		gha.Infof("Stopping buildkitd...")
		err = bkCli.Stop(ctx)
		if err != nil {
			gha.Errorf("Failed to stop buildkitd container: %+v", err)
			return
		}

		var compressionLevel int
		compressionLevel, err = strconv.Atoi(gha.GetInput(inputCompressionLevel))
		if err != nil {
			gha.Errorf(`Failed to parse "%s": %+v`, inputCompressionLevel, err)
			err = errors.WithStack(err)
			return
		}

		gha.Infof("Extract and compress buildkit state...")
		var buf *bytes.Buffer
		buf, err = CompressToZstd(ctx, bkCli, compressionLevel)
		if err != nil {
			gha.Errorf("Failed to compress buildkit state: %+v", err)
			return
		}

		gha.Infof("Uploading to remote storage...")
		err = manager.Save(ctx, cacheKey, buf.Bytes())
		if err != nil {
			gha.Errorf("Failed to save compressed buildkit sate to remote: %+v", err)
			return
		}
	}()

	return err
}
