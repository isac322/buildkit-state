package remote

import (
	"context"
	"io"

	"github.com/samber/mo"
)

type LoadedCache struct {
	Key   string
	Data  io.ReadCloser
	Extra map[string]any
}

type Manager interface {
	Load(ctx context.Context, primaryKey string, secondaryKeys []string) (mo.Option[LoadedCache], error)
	Save(ctx context.Context, cacheKey string, data []byte) error
}
