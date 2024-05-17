package node

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

var defaultColumnGroups = []string{"labels", "capacity"}
var orderByFields = []string{"id", "type", "available_cpu", "available_memory", "available_disk", "available_gpu", "status"}
var filterApprovalValues = []string{"approved", "pending", "rejected"}
var filterStatusValues = []string{"connected", "disconnected"}

// ListOptions is a struct to support node command
type ListOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	ColumnGroups     []string
	Labels           string
	FilterByApproval string
	FilterByStatus   string
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
		Use:   "list",
		Short: "List info of network nodes. ",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig()
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, api)
		},
	}
	nodeCmd.Flags().StringSliceVar(&o.ColumnGroups, "show", o.ColumnGroups,
		fmt.Sprintf("What column groups to show. Zero or more of: %q", maps.Keys(toggleColumns)))
	nodeCmd.Flags().StringVar(&o.Labels, "labels", o.Labels,
		"Filter nodes by labels. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more information.")
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	nodeCmd.Flags().StringVar(&o.FilterByApproval, "filter-approval", o.FilterByApproval,
		fmt.Sprintf("Filter nodes by approval. One of: %q", filterApprovalValues))
	nodeCmd.Flags().StringVar(&o.FilterByStatus, "filter-status", o.FilterByStatus,
		fmt.Sprintf("Filter nodes by status. One of: %q", filterStatusValues))

	return nodeCmd
}

// Run executes node command
func (o *ListOptions) run(cmd *cobra.Command, api client.API) error {
	ctx := cmd.Context()

	var err error
	var labelRequirements []labels.Requirement
	if o.Labels != "" {
		labelRequirements, err = labels.ParseToRequirements(o.Labels)
		if err != nil {
			return fmt.Errorf("could not parse labels: %w", err)
		}
	}

	if o.FilterByApproval != "" {
		if !slices.Contains(filterApprovalValues, o.FilterByApproval) {
			return fmt.Errorf("cannot use '%s' as filter-approval value, should be one of: %q", o.FilterByApproval, filterApprovalValues)
		}
	}

	if o.FilterByStatus != "" {
		if !slices.Contains(filterStatusValues, o.FilterByStatus) {
			return fmt.Errorf("cannot use '%s' as filter-status value, should be one of: %q", o.FilterByStatus, filterStatusValues)
		}
	}

	response, err := api.Nodes().List(ctx, &apimodels.ListNodesRequest{
		Labels:           labelRequirements,
		FilterByApproval: o.FilterByApproval,
		FilterByStatus:   o.FilterByStatus,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		return fmt.Errorf("failed request: %w", err)
	}

	columns := alwaysColumns
	for _, label := range o.ColumnGroups {
		columns = append(columns, toggleColumns[label]...)
	}

	if err = output.Output(cmd, columns, o.OutputOptions, response.Nodes); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
