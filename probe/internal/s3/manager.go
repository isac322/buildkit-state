package s3

import (
	"bytes"
	"context"
	"path"
	"path/filepath"

	"github.com/isac322/buildkit-state/probe/internal"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
	"github.com/samber/mo"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

const version = "v1"

type Manager struct {
	client    *s3.Client
	bucket    string
	keyPrefix string
}

func New(cfg aws.Config, bucket, keyPrefix string, s3Compatible bool) Manager {
	client := s3.NewFromConfig(
		cfg,
		func(options *s3.Options) {
			if s3Compatible {
				options.UsePathStyle = true
			}
		},
	)
	return Manager{client, bucket, keyPrefix}
}

func (m Manager) Load(
	ctx context.Context,
	primaryKey string,
	secondaryKeys []string,
) (mo.Option[internal.LoadedCache], error) {
	result, err := m.loadLatestOrExactMatch(ctx, primaryKey, secondaryKeys)
	if err != nil {
		return mo.None[internal.LoadedCache](), err
	}
	metadata, found := result.Get()
	if !found {
		return mo.None[internal.LoadedCache](), nil
	}

	object, err := m.client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: &m.bucket,
			Key:    metadata.Key,
		},
	)
	if err != nil {
		return mo.None[internal.LoadedCache](), errors.WithStack(err)
	}

	rel, err := filepath.Rel(version, *metadata.Key)
	if err != nil {
		return mo.None[internal.LoadedCache](), errors.Wrap(err, "version does not match")
	}
	return mo.Some(internal.LoadedCache{
		Key:   rel,
		Data:  object.Body,
		Extra: nil,
	}), nil
}

func (m Manager) loadLatestOrExactMatch(
	ctx context.Context,
	primaryKey string,
	secondaryKeys []string,
) (mo.Option[types.Object], error) {
	errGrp, grpCtx := errgroup.WithContext(ctx)

	c := make(chan *s3.ListObjectsV2Output)

	keys := make([]string, 0, 1+len(secondaryKeys))
	keys = append(keys, m.buildS3Key(primaryKey))
	for _, key := range secondaryKeys {
		keys = append(keys, m.buildS3Key(key))
	}

	for _, key := range keys {
		key := key
		errGrp.Go(func() error {
			paginator := s3.NewListObjectsV2Paginator(
				m.client,
				&s3.ListObjectsV2Input{
					Bucket: &m.bucket,
					Prefix: &key,
				},
			)
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(grpCtx)
				if err != nil {
					return errors.WithStack(err)
				}

				select {
				case c <- page:
				case <-grpCtx.Done():
					return grpCtx.Err()
				}
			}
			return nil
		})
	}

	var err error
	go func() {
		err = errGrp.Wait()
		close(c)
	}()

	var result mo.Option[types.Object]
	exactMatchFound := false
	for obj := range c {
		if exactMatchFound {
			continue
		}

		for _, content := range obj.Contents {
			if slices.Contains(keys, *content.Key) {
				exactMatchFound = true
				result = mo.Some(content)
				break
			}

			if prev, found := result.Get(); found {
				if prev.LastModified.IsZero() || prev.LastModified.Before(*content.LastModified) {
					result = mo.Some(content)
				}
			} else {
				result = mo.Some(content)
			}
		}
	}
	if err != nil {
		return mo.None[types.Object](), errors.WithStack(err)
	}

	return result, nil
}

func (m Manager) Save(ctx context.Context, cacheKey string, data []byte) error {
	key := m.buildS3Key(cacheKey)
	_, err := m.client.PutObject(
		ctx,
		&s3.PutObjectInput{Bucket: &m.bucket, Key: &key, Body: bytes.NewReader(data)},
	)
	return errors.WithStack(err)
}

func (m Manager) buildS3Key(key string) string {
	return path.Join(version, m.keyPrefix, key)
}

var (
	_ internal.Loader = Manager{}
	_ internal.Saver  = Manager{}
)
