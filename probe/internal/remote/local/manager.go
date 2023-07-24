package localmanager

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isac322/buildkit-state/probe/internal/remote"

	"github.com/pkg/errors"
	"github.com/samber/mo"
	"golang.org/x/exp/slices"
)

const version = "v1"

type Manager struct {
	dest string
}

func New(destinationPath string) Manager {
	return Manager{destinationPath}
}

type entry struct {
	FullPath string
	ModTime  time.Time
}

func (m Manager) Load(
	_ context.Context,
	primaryKey string,
	secondaryKeys []string,
) (mo.Option[remote.LoadedCache], error) {
	keys := make([]string, 0, 1+len(secondaryKeys))
	keys = append(keys, primaryKey)
	keys = append(keys, secondaryKeys...)

	var matched mo.Option[entry]

	err := filepath.WalkDir(
		filepath.Join(m.dest, version),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}
			if d.IsDir() {
				return nil
			}
			filename := d.Name()

			// exact match
			if slices.Contains(keys, filename) {
				info, err := d.Info()
				if err != nil {
					return errors.WithStack(err)
				}

				matched = mo.Some(entry{FullPath: path, ModTime: info.ModTime()})
				return filepath.SkipAll
			}

			// prefix match
			if slices.ContainsFunc(
				keys,
				func(s string) bool {
					return strings.HasPrefix(filename, s)
				},
			) {
				info, err := d.Info()
				if err != nil {
					return errors.WithStack(err)
				}

				if prev, exist := matched.Get(); exist && prev.ModTime.After(info.ModTime()) {
					return nil
				}

				matched = mo.Some(entry{FullPath: path, ModTime: info.ModTime()})
				return nil
			}

			return nil
		},
	)
	if errors.Is(err, os.ErrNotExist) {
		return mo.None[remote.LoadedCache](), nil
	}
	if err != nil {
		return mo.None[remote.LoadedCache](), err
	}

	fileInfo, found := matched.Get()
	if !found {
		return mo.None[remote.LoadedCache](), nil
	}

	fullPath := fileInfo.FullPath
	fp, err := os.Open(fullPath)
	if err != nil {
		return mo.None[remote.LoadedCache](), errors.WithStack(err)
	}

	return mo.Some(remote.LoadedCache{
		Key:   filepath.Base(fullPath),
		Data:  fp,
		Extra: nil,
	}), err
}

func (m Manager) Save(_ context.Context, cacheKey string, data []byte) error {
	err := os.MkdirAll(filepath.Join(m.dest, version), os.ModePerm)
	if err != nil {
		return errors.WithStack(err)
	}

	fp, err := os.OpenFile(filepath.Join(m.dest, version, cacheKey), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o660)
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err = fp.Write(data); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(fp.Close())
}

var _ remote.Manager = Manager{}
