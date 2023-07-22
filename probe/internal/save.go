package internal

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	bkclient "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/exp/slices"
)

type Saver interface {
	Save(ctx context.Context, cacheKey string, data []byte) error
}

func SaveFromContainerToRemote(
	ctx context.Context,
	gha *githubactions.Action,
	docker client.CommonAPIClient,
	buildkit *bkclient.Client,
	builderName string,
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

		targetTypes := getMultilineInput(gha, inputTargetTypes)
		filters := make([]string, 0, len(pruneTypes))
		for _, recordType := range pruneTypes {
			rt := string(recordType)
			if slices.Contains(targetTypes, rt) {
				continue
			}
			filters = append(filters, "type=="+rt)
		}

		err = buildkit.Prune(ctx, nil, bkclient.WithFilter(filters))
		if err != nil {
			gha.Errorf(`Failed to prune caches: %+v`, err)
			return
		}

		err = printDiskUsage(ctx, gha, buildkit)
	}()
	if err != nil {
		return err
	}

	containerName := BuildKitContainerNameFromBuilder(builderName)

	func() {
		gha.Group("Save buildkit state to remote")
		defer gha.EndGroup()

		gha.Infof("stopping buildkitd...")
		err = docker.ContainerStop(ctx, containerName, container.StopOptions{})
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
		buf, err = CompressToZstd(ctx, docker, containerName, compressionLevel)
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
