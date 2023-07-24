package buildkit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	bkclient "github.com/moby/buildkit/client"
	pkgerrors "github.com/pkg/errors"
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

type commonDriver struct {
	bkClient *bkclient.Client
}

func newCommonDriver(bkClient *bkclient.Client) commonDriver {
	return commonDriver{bkClient: bkClient}
}

func (d *commonDriver) PruneExcept(ctx context.Context, whitelist []string) error {
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

func (d *commonDriver) PrintDiskUsage(ctx context.Context) ([]byte, error) {
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
		return nil, pkgerrors.WithStack(err)
	}

	return buf.Bytes(), nil
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
