// Package buildkitdial provides connhelper for docker-container://<container>
package buildkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	bkclient "github.com/moby/buildkit/client"
	pkgerrors "github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type dockerContainer struct {
	docker        client.CommonAPIClient
	builderName   string
	containerName string
	commonDriver
}

func NewContainerizedDriver(ctx context.Context, docker client.CommonAPIClient, builderName string) (Driver, error) {
	containerName := fmt.Sprintf("buildx_buildkit_%s0", builderName)
	bkcli, err := bkclient.New(
		ctx,
		"",
		bkclient.WithContextDialer(newContainerDialer(docker, containerName)),
	)
	if err != nil {
		return nil, err
	}

	return &dockerContainer{
		docker,
		builderName,
		containerName,
		newCommonDriver(bkcli),
	}, nil
}

func (d *dockerContainer) Stop(ctx context.Context) error {
	return d.docker.ContainerStop(ctx, d.containerName, container.StopOptions{})
}

func (d *dockerContainer) Resume(ctx context.Context) error {
	err := d.docker.ContainerStart(ctx, d.containerName, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	bkcli, err := bkclient.New(
		ctx,
		"",
		bkclient.WithContextDialer(newContainerDialer(d.docker, d.containerName)),
	)
	if err != nil {
		return err
	}

	d.commonDriver.bkClient = bkcli
	return nil
}

func newContainerDialer(
	docker client.ContainerAPIClient,
	containerName string,
) func(ctx context.Context, _ string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		exec, err := docker.ContainerExecCreate(
			ctx,
			containerName,
			types.ExecConfig{
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Cmd:          []string{"buildctl", "dial-stdio"},
			},
		)
		if err != nil {
			return nil, err
		}

		attach, err := docker.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{})
		if err != nil {
			return nil, err
		}
		return newHijackedNetConn(attach), nil
	}
}

func (d *dockerContainer) CopyFrom(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	contents, stats, err := d.docker.CopyFromContainer(ctx, d.containerName, path)
	if err != nil {
		return nil, 0, err
	}

	return contents, stats.Size, nil
}

func (d *dockerContainer) CopyTo(ctx context.Context, path string, content io.Reader) error {
	return d.docker.CopyToContainer(
		ctx,
		d.containerName,
		path,
		content,
		types.CopyToContainerOptions{AllowOverwriteDirWithFile: true},
	)
}

type hijackedNetConn struct {
	net.Conn
	errGrp        *errgroup.Group
	decodedStdOut io.ReadCloser
}

func newHijackedNetConn(res types.HijackedResponse) *hijackedNetConn {
	errGrp := new(errgroup.Group)
	decodedStdOut, pipeWriter := io.Pipe()
	errGrp.Go(func() error {
		_, err := stdcopy.StdCopy(pipeWriter, io.Discard, res.Conn)
		return err
	})

	return &hijackedNetConn{
		res.Conn,
		errGrp,
		decodedStdOut,
	}
}

func (h hijackedNetConn) Read(b []byte) (n int, err error) {
	read, err := h.decodedStdOut.Read(b)
	if err != nil {
		return read, pkgerrors.WithStack(err)
	}
	return read, err
}

func (h hijackedNetConn) Close() error {
	connErr := pkgerrors.WithStack(h.Conn.Close())
	grpErr := pkgerrors.WithStack(h.errGrp.Wait())

	if connErr != nil && grpErr != nil {
		return errors.Join(connErr, grpErr)
	} else if connErr != nil {
		return connErr
	} else if grpErr != nil {
		return grpErr
	}
	return nil
}

var _ net.Conn = hijackedNetConn{}
