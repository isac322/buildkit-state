package internal

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/isac322/buildkit-state/probe/internal/buildkit"

	pkgerrors "github.com/pkg/errors"
	"github.com/valyala/gozstd"
)

const is64Bit = uint64(^uintptr(0)) == ^uint64(0)

func DecompressZstdTo(ctx context.Context, bkCli buildkit.Driver, body io.ReadCloser) (err error) {
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

	return bkCli.CopyTo(ctx, BuildKitStateDir, zstdReader)
}

func CompressToZstd(ctx context.Context, bkCli buildkit.Driver, compressionLevel int) (*bytes.Buffer, error) {
	contents, size, err := bkCli.CopyFrom(ctx, BuildKitStateDir)
	if err != nil {
		return nil, err
	}

	var windowLog int
	if is64Bit {
		windowLog = gozstd.WindowLogMax64
	} else {
		windowLog = gozstd.WindowLogMax32
	}

	buf := bytes.NewBuffer(make([]byte, 0, size/2))
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
