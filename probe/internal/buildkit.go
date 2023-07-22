package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-githubactions"
	"github.com/tonistiigi/units"
)

func printKV(w io.Writer, k string, v any) {
	fmt.Fprintf(w, "%s:\t%v\n", k, v)
}

func printVerbose(tw *tabwriter.Writer, du []*client.UsageInfo) {
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

func printSummary(tw *tabwriter.Writer, du []*client.UsageInfo) {
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

func printDiskUsage(ctx context.Context, gha *githubactions.Action, cli *client.Client) error {
	gha.Infof("printing disk usage...")
	var usages []*client.UsageInfo
	usages, err := cli.DiskUsage(ctx)
	if err != nil {
		gha.Errorf("Failed to get disk usage of buildkitd: %+v", err)
		return err
	}

	buf := new(bytes.Buffer)
	tw := tabwriter.NewWriter(buf, 1, 8, 1, '\t', 0)
	printVerbose(tw, usages)
	printSummary(tw, usages)
	if err = tw.Flush(); err != nil {
		gha.Errorf("Failed to print disk usage: %+v", err)
		return errors.WithStack(err)
	}
	gha.Infof(buf.String())
	return nil
}
