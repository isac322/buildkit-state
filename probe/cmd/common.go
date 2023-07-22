package main

import (
	"context"

	"github.com/isac322/buildkit-state/probe/internal"
	"github.com/isac322/buildkit-state/probe/internal/github"
	"github.com/isac322/buildkit-state/probe/internal/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/docker/docker/client"
	bkclient "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
)

const (
	inputBuildxName   = "buildx-name"
	inputRemoteType   = "remote-type"
	inputS3BucketName = "s3-bucket-name"
	inputS3KeyPrefix  = "s3-key-prefix"
	inputS3URL        = "s3-url"
)

func newManager(ctx context.Context, gha *githubactions.Action) (manager internal.RemoteManager, err error) {
	remoteType := gha.GetInput(inputRemoteType)

	switch remoteType {
	case "gha":
		manager, err = github.New()
		if err != nil {
			gha.Errorf("Failed to access Github Actions Cache: %+v", err)
			return nil, err
		}
		return manager, nil

	case "s3":
		bucketName := gha.GetInput(inputS3BucketName)
		if bucketName == "" {
			err = errors.Errorf(`"%s" is required`, inputS3BucketName)
			gha.Errorf(err.Error())
			return nil, err
		}
		keyPrefix := gha.GetInput(inputS3KeyPrefix)
		customURL := gha.GetInput(inputS3URL)

		var opts []func(*config.LoadOptions) error
		if customURL != "" {
			opts = append(opts, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...any) (aws.Endpoint, error) {
					return aws.Endpoint{URL: customURL}, nil
				},
			)))
		}

		awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			gha.Errorf("Failed to load aws config: %+v", err)
			return nil, err
		}

		return s3.New(awsCfg, bucketName, keyPrefix, customURL != ""), nil

	default:
		err = errors.Errorf("unknown remote-type: %v. Only supports `gha` or `s3`", remoteType)
		gha.Errorf(err.Error())
		return nil, err
	}
}

func newDependencies(ctx context.Context) (
	*githubactions.Action,
	client.CommonAPIClient,
	*bkclient.Client,
	string,
	internal.RemoteManager,
	error,
) {
	gha := githubactions.New()
	manager, err := newManager(ctx, gha)
	if err != nil {
		return nil, nil, nil, "", nil, err
	}

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		gha.Errorf("Failed connect docker: %+v", err)
		return nil, nil, nil, "", nil, err
	}

	builderName := gha.GetInput(inputBuildxName)
	bkcli, err := bkclient.New(
		ctx,
		"docker-container://"+internal.BuildKitContainerNameFromBuilder(builderName),
	)
	if err != nil {
		gha.Errorf(
			"Failed connect buildkitd: %+v. The Action only supports `docker-container` driver of buildkit",
			err,
		)
		return nil, nil, nil, "", nil, err
	}

	return gha, docker, bkcli, builderName, manager, nil
}
