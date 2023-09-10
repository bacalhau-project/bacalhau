package node

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"
)

// DescribeOptions is a struct to support node command
type DescribeOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewDescribeOptions returns initialized Options
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{
		OutputOpts: output.NonTabularOutputOptions{Format: output.YAMLFormat},
	}
}

func NewDescribeCmd() *cobra.Command {
	o := NewDescribeOptions()
	nodeCmd := &cobra.Command{
		Use:   "describe [id]",
		Short: "Get the info of a node by id.",
		Args:  cobra.ExactArgs(1),
		Run:   o.runDescribe,
	}
	nodeCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return nodeCmd
}

// Run executes node command
func (o *DescribeOptions) runDescribe(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	nodeID := args[0]
	response, err := util.GetAPIClientV2(ctx).Nodes().Get(&apimodels.GetNodeRequest{
		NodeID: nodeID,
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not get node %s: %w", nodeID, err), 1)
	}

	if err = output.OutputOneNonTabular(cmd, o.OutputOpts, response.Node); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write node %s: %w", nodeID, err), 1)
	}
}
