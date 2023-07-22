package internal

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	pkgerrors "github.com/pkg/errors"
	"github.com/valyala/gozstd"
)

const is64Bit = uint64(^uintptr(0)) == ^uint64(0)

func DecompressZstdTo(
	ctx context.Context,
	docker client.CommonAPIClient,
	container string,
	body io.ReadCloser,
) (err error) {
	zstdReader := gozstd.NewReader(body)
	defer zstdReader.Release()
	defer func() {
		closeErr := pkgerrors.WithStack(body.Close())

		if closeErr != nil {
			if err != nil {
				err = errors.Join(err, closeErr)
			} else {
				err = closeErr
			}
		}
	}()

	return docker.CopyToContainer(ctx, container, BuildKitStateDir, zstdReader, types.CopyToContainerOptions{})
}

func CompressToZstd(
	ctx context.Context,
	docker client.CommonAPIClient,
	container string,
	compressionLevel int,
) (*bytes.Buffer, error) {
	contents, stats, err := docker.CopyFromContainer(ctx, container, BuildKitStateDir)
	if err != nil {
		return nil, err
	}

	var windowLog int
	if is64Bit {
		windowLog = gozstd.WindowLogMax64
	} else {
		windowLog = gozstd.WindowLogMax32
	}

	buf := bytes.NewBuffer(make([]byte, 0, stats.Size/2))
	writer := gozstd.NewWriterParams(
		buf,
		&gozstd.WriterParams{
			CompressionLevel: compressionLevel,
			WindowLog:        windowLog,
		},
	)
	defer writer.Release()
	_, err = writer.ReadFrom(contents)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	if err = writer.Close(); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	if err = contents.Close(); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return buf, nil
}
