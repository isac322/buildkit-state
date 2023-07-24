package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"

	"github.com/isac322/buildkit-state/probe/internal/buildkit"

	"github.com/klauspost/compress/zstd"
	pkgerrors "github.com/pkg/errors"
)

func DecompressZstdTo(ctx context.Context, bkCli buildkit.Driver, body io.ReadCloser) error {
	reader, err := zstd.NewReader(
		body,
		zstd.WithDecoderLowmem(false),
		zstd.WithDecoderConcurrency(runtime.GOMAXPROCS(0)),
	)
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	return bkCli.CopyTo(ctx, BuildKitStateLoadDir, reader.IOReadCloser())
}

func CompressToZstd(ctx context.Context, bkCli buildkit.Driver, compressionLevel int) (buf *bytes.Buffer, err error) {
	contents, size, err := bkCli.CopyFrom(ctx, BuildKitStateSaveDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := contents.Close()
		if closeErr != nil {
			if err != nil {
				err = errors.Join(err, closeErr)
			} else {
				err = pkgerrors.WithStack(closeErr)
			}
		}
	}()

	buf = bytes.NewBuffer(make([]byte, 0, size/2))
	writer, err := zstd.NewWriter(
		buf,
		zstd.WithEncoderConcurrency(runtime.GOMAXPROCS(0)),
		zstd.WithNoEntropyCompression(false),
		zstd.WithWindowSize(zstd.MaxWindowSize),
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)),
	)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	_, err = writer.ReadFrom(contents)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	if err = writer.Close(); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return buf, nil
}
