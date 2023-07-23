package s3manager

import (
	"bytes"
	"context"
	"math/rand"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/caarlos0/env/v9"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Host      string `env:"S3_HOST" envDefault:"localhost"`
	Port      int    `env:"S3_PORT"`
	Region    string `env:"S3_REGION" envDefault:"us-west-2"`
	AccessKey string `env:"S3_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"S3_SECRET_KEY" envDefault:"minioadmin"`
}

func newConfig() (cfg Config, err error) {
	err = errors.WithStack(env.Parse(&cfg))
	return
}

func newAWSConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := newConfig()
	if err != nil {
		return aws.Config{}, errors.WithStack(err)
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{
				// PartitionID:   "aws",
				URL: "http://" + net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
				// SigningRegion: "us-west-2",
			}, nil
		},
	)

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithLogger(logging.NewStandardLogger(os.Stdout)),
	)
	if err != nil {
		return aws.Config{}, errors.WithStack(err)
	}

	return awsCfg, nil
}

func TestManager_LoadTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		predefinedKeys []string
		keyPrefix      string
		primaryKey     string
		secondaryKeys  []string
		found          bool
		expectedKey    string
	}{
		{
			name:           "empty",
			predefinedKeys: nil,
			keyPrefix:      "",
			primaryKey:     "",
			secondaryKeys:  nil,
			found:          false,
			expectedKey:    "",
		},
		{
			name:           "single exact matched object",
			predefinedKeys: []string{"v1/key-with-dashes"},
			keyPrefix:      "",
			primaryKey:     "key-with-dashes",
			secondaryKeys:  nil,
			found:          true,
			expectedKey:    "key-with-dashes",
		},
		{
			name:           "single exact matched object with prefix",
			predefinedKeys: []string{"v1/prefixed/key-with-dashes"},
			keyPrefix:      "prefixed",
			primaryKey:     "key-with-dashes",
			secondaryKeys:  nil,
			found:          true,
			expectedKey:    "prefixed/key-with-dashes",
		},
		{
			name:           "single exact matched from secondary",
			predefinedKeys: []string{"v1/prefixed/key-with-dashes"},
			keyPrefix:      "prefixed",
			primaryKey:     "does-not-exists",
			secondaryKeys:  []string{"key-with-dashes"},
			found:          true,
			expectedKey:    "prefixed/key-with-dashes",
		},
		{
			name:           "single prefixed matched from secondary",
			predefinedKeys: []string{"v1/prefixed/key-with-dashes-and-extra"},
			keyPrefix:      "prefixed",
			primaryKey:     "does-not-exists",
			secondaryKeys:  []string{"key-with-dashes"},
			found:          true,
			expectedKey:    "prefixed/key-with-dashes-and-extra",
		},
		{
			name: "multiple matches",
			predefinedKeys: []string{
				"v1/prefixed/key-with-dashes-oldest",
				"v1/prefixed/key-with-dashes-0",
				"v1/prefixed/key-with-dashes-1",
				"v1/prefixed/key-with-dashes-newest",
			},
			keyPrefix:     "prefixed",
			primaryKey:    "does-not-exists",
			secondaryKeys: []string{"key-with-dashes"},
			found:         true,
			expectedKey:   "prefixed/key-with-dashes-newest",
		},
		{
			name: "multiple matches - prefer exact match",
			predefinedKeys: []string{
				"v1/prefixed/key-with-dashes-oldest",
				"v1/prefixed/key-with-dashes-0",
				"v1/prefixed/key-with-dashes",
				"v1/prefixed/key-with-dashes-1",
				"v1/prefixed/key-with-dashes-newest",
			},
			keyPrefix:     "prefixed",
			primaryKey:    "does-not-exists",
			secondaryKeys: []string{"key-with-dashes"},
			found:         true,
			expectedKey:   "prefixed/key-with-dashes",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			awsConfig, err := newAWSConfig(ctx)
			require.NoError(t, err)

			bucket := strconv.Itoa(rand.Int()) // nolint:gosec
			manager := New(awsConfig, bucket, tc.keyPrefix, true)

			_, err = manager.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &bucket})
			require.NoError(t, err)

			for i, key := range tc.predefinedKeys {
				key := key
				if i != 0 {
					time.Sleep(100 * time.Millisecond)
				}
				_, err = manager.client.PutObject(
					ctx,
					&s3.PutObjectInput{
						Bucket: &bucket,
						Key:    &key,
						Body:   bytes.NewReader([]byte("data")),
					},
				)
				require.NoError(t, errors.WithStack(err))
			}

			result, err := manager.Load(ctx, tc.primaryKey, tc.secondaryKeys)
			assert.NoError(t, err)
			cache, found := result.Get()
			assert.Equal(t, tc.found, found)
			assert.Equal(t, tc.expectedKey, cache.Key)
		})
	}
}
