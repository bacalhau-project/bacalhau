package node

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/labels"
)

var defaultColumnGroups = []string{"labels", "capacity"}
var orderByFields = []string{"id", "type", "available_cpu", "available_memory", "available_disk", "available_gpu"}

// ListOptions is a struct to support node command
type ListOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	ColumnGroups []string
	Labels       string
}

// NewListOptions returns initialized Options
func NewListOptions() *ListOptions {
	return &ListOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		ListOptions:   cliflags.ListOptions{OrderByFields: orderByFields},
		ColumnGroups:  defaultColumnGroups,
	}
}

func NewListCmd() *cobra.Command {
	o := NewListOptions()
	nodeCmd := &cobra.Command{
		Use:    "list",
		Short:  "List info of network nodes. ",
		PreRun: util.ApplyPorcelainLogLevel,
		Args:   cobra.NoArgs,
		Run:    o.run,
	}
	nodeCmd.Flags().StringSliceVar(&o.ColumnGroups, "show", o.ColumnGroups,
		fmt.Sprintf("What column groups to show. Zero or more of: %q", maps.Keys(toggleColumns)))

	nodeCmd.Flags().StringVar(&o.Labels, "labels", o.Labels,
		"Filter nodes by labels. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more information.")
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

// Run executes node command
func (o *ListOptions) run(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	var err error
	var labelRequirements []labels.Requirement
	if o.Labels != "" {
		labelRequirements, err = labels.ParseToRequirements(o.Labels)
		if err != nil {
			util.Fatal(cmd, fmt.Errorf("could not parse labels: %w", err), 1)
		}
	}
	response, err := util.GetAPIClientV2(ctx).Nodes().List(&apimodels.ListNodesRequest{
		Labels: labelRequirements,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("failed request: %w", err), 1)
	}

	columns := alwaysColumns
	for _, label := range o.ColumnGroups {
		columns = append(columns, toggleColumns[label]...)
	}

	if err = output.Output(cmd, columns, o.OutputOptions, response.Nodes); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to output: %w", err), 1)
	}
}
