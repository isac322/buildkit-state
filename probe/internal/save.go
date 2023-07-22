package internal

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/isac322/buildkit-state/probe/internal/buildkit"
	gha2 "github.com/isac322/buildkit-state/probe/internal/gha"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
)

type Saver interface {
	Save(ctx context.Context, cacheKey string, data []byte) error
}

func SaveFromContainerToRemote(
	ctx context.Context,
	gha *githubactions.Action,
	bkCli buildkit.Driver,
	saver Saver,
) (err error) {
	cacheKey := gha.GetInput(inputPrimaryKey)
	restoredCacheKey := gha.Getenv("STATE_" + strings.ToUpper(strings.ReplaceAll(stateLoadedCacheKey, " ", "_")))

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

		gha.Infof("stopping buildkitd...")
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

		var buf *bytes.Buffer
		buf, err = CompressToZstd(ctx, bkCli, compressionLevel)
		if err != nil {
			gha.Errorf("Failed to compress buildkit state: %+v", err)
			return
		}

		err = saver.Save(ctx, cacheKey, buf.Bytes())
		if err != nil {
			gha.Errorf("Failed to save compressed buildkit sate to remote: %+v", err)
			return
		}
	}()

	return err
}
