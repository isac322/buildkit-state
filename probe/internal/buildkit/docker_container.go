// Package buildkitdial provides connhelper for docker-container://<container>
package buildkit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	bkclient "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/tonistiigi/units"
	"golang.org/x/exp/slices"
)

var pruneTypes = []bkclient.UsageRecordType{
	bkclient.UsageRecordTypeInternal,
	bkclient.UsageRecordTypeFrontend,
	bkclient.UsageRecordTypeLocalSource,
	bkclient.UsageRecordTypeGitCheckout,
	bkclient.UsageRecordTypeCacheMount,
	bkclient.UsageRecordTypeRegular,
}

type dockerContainer struct {
	docker        client.CommonAPIClient
	builderName   string
	containerName string
	bkClient      *bkclient.Client
}

func NewContainerizedDriver(docker client.CommonAPIClient, builderName string) Driver {
	d := &dockerContainer{
		docker,
		builderName,
		fmt.Sprintf("buildx_buildkit_%s0", builderName),
		nil,
	}
	return d
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
		bkclient.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			exec, err := d.docker.ContainerExecCreate(
				ctx,
				d.containerName,
				types.ExecConfig{AttachStdin: true, Cmd: []string{"buildctl", "dial-stdio"}},
			)
			if err != nil {
				return nil, err
			}

			attach, err := d.docker.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{})
			if err != nil {
				return nil, err
			}

			return attach.Conn, nil
		}),
	)
	if err != nil {
		return err
	}

	d.bkClient = bkcli
	return nil
}

func (d *dockerContainer) PruneExcept(ctx context.Context, whitelist []string) error {
	filters := make([]string, 0, len(pruneTypes))
	for _, recordType := range pruneTypes {
		rt := string(recordType)
		if slices.Contains(whitelist, rt) {
			continue
		}
		filters = append(filters, "type=="+rt)
	}

	return d.bkClient.Prune(ctx, nil, bkclient.WithFilter(filters))
}

func (d *dockerContainer) PrintDiskUsage(ctx context.Context) ([]byte, error) {
	var usages []*bkclient.UsageInfo
	usages, err := d.bkClient.DiskUsage(ctx)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	tw := tabwriter.NewWriter(buf, 1, 8, 1, '\t', 0)
	printVerbose(tw, usages)
	printSummary(tw, usages)
	if err = tw.Flush(); err != nil {
		return nil, errors.WithStack(err)
	}

	return buf.Bytes(), nil
}

func (d *dockerContainer) CopyFrom(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	contents, stats, err := d.docker.CopyFromContainer(ctx, d.containerName, path)
	if err != nil {
		return nil, 0, err
	}

	return contents, stats.Size, nil
}

func (d *dockerContainer) CopyTo(ctx context.Context, path string, content io.Reader) error {
	return d.docker.CopyToContainer(ctx, d.containerName, path, content, types.CopyToContainerOptions{})
}

func printKV(w io.Writer, k string, v any) {
	fmt.Fprintf(w, "%s:\t%v\n", k, v)
}

func printVerbose(tw *tabwriter.Writer, du []*bkclient.UsageInfo) {
	for _, di := range du {
		printKV(tw, "ID", di.ID)
		if len(di.Parents) > 0 {
			printKV(tw, "Parents", strings.Join(di.Parents, ";"))
		}
		printKV(tw, "Created at", di.CreatedAt)
		printKV(tw, "Mutable", di.Mutable)
		printKV(tw, "Reclaimable", !di.InUse)
		printKV(tw, "Shared", di.Shared)
		printKV(tw, "Size", fmt.Sprintf("%.2f", units.Bytes(di.Size)))
		if di.Description != "" {
			printKV(tw, "Description", di.Description)
		}
		printKV(tw, "Usage count", di.UsageCount)
		if di.LastUsedAt != nil {
			printKV(tw, "Last used", di.LastUsedAt)
		}
		if di.RecordType != "" {
			printKV(tw, "Type", di.RecordType)
		}

		fmt.Fprintf(tw, "\n")
	}

	tw.Flush()
}

func printSummary(tw *tabwriter.Writer, du []*bkclient.UsageInfo) {
	total := int64(0)
	reclaimable := int64(0)
	shared := int64(0)

	for _, di := range du {
		if di.Size > 0 {
			total += di.Size
			if !di.InUse {
				reclaimable += di.Size
			}
		}
		if di.Shared {
			shared += di.Size
		}
	}

	if shared > 0 {
		fmt.Fprintf(tw, "Shared:\t%.2f\n", units.Bytes(shared))
		fmt.Fprintf(tw, "Private:\t%.2f\n", units.Bytes(total-shared))
	}

	fmt.Fprintf(tw, "Reclaimable:\t%.2f\n", units.Bytes(reclaimable))
	fmt.Fprintf(tw, "Total:\t%.2f\n", units.Bytes(total))
	tw.Flush()
}
