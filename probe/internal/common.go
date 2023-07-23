package internal

import (
	"context"

	"github.com/samber/mo"
)

type RemoteManager interface {
	Load(ctx context.Context, primaryKey string, secondaryKeys []string) (mo.Option[LoadedCache], error)
	Save(ctx context.Context, cacheKey string, data []byte) error
}
