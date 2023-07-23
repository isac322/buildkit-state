package github

import (
	"context"
	"io"

	"github.com/isac322/buildkit-state/probe/internal"

	"github.com/pkg/errors"
	"github.com/samber/mo"
	actionscache "github.com/tonistiigi/go-actions-cache"
)

type Manager struct {
	gha *actionscache.Cache
}

func New() (Manager, error) {
	gha, err := actionscache.TryEnv(actionscache.Opt{})
	if err != nil {
		return Manager{}, errors.WithStack(err)
	}
	return Manager{gha}, nil
}

func (m Manager) Load(
	ctx context.Context,
	primaryKey string,
	secondaryKeys []string,
) (mo.Option[internal.LoadedCache], error) {
	keys := make([]string, 0, 1+len(secondaryKeys))
	keys = append(keys, primaryKey)
	keys = append(keys, secondaryKeys...)
	cache, err := m.gha.Load(ctx, keys...)
	if err != nil {
		return mo.None[internal.LoadedCache](), errors.WithStack(err)
	}
	if cache == nil {
		return mo.None[internal.LoadedCache](), nil
	}

	return mo.Some(internal.LoadedCache{
		Key:   cache.Key,
		Data:  &wrappedBody{cache.Download(ctx), 0},
		Extra: nil,
	}), nil
}

func (m Manager) Save(ctx context.Context, cacheKey string, data []byte) error {
	return errors.WithStack(m.gha.Save(ctx, cacheKey, actionscache.NewBlob(data)))
}

var _ internal.RemoteManager = Manager{}

type wrappedBody struct {
	actionscache.ReaderAtCloser
	offset int
}

func (r *wrappedBody) Read(b []byte) (int, error) {
	n, err := r.ReadAt(b, int64(r.offset))
	r.offset += n
	if n > 0 && err == io.EOF {
		err = nil
	}
	return n, err
}
