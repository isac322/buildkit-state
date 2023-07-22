package buildkit

import (
	"context"
	"io"
)

type Driver interface {
	Stop(ctx context.Context) error
	Resume(ctx context.Context) error
	PruneExcept(ctx context.Context, whitelist []string) error
	PrintDiskUsage(ctx context.Context) ([]byte, error)
	CopyFrom(ctx context.Context, path string) (io.ReadCloser, int64, error)
	CopyTo(ctx context.Context, path string, content io.Reader) error
}
