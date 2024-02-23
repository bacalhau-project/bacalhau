package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// NodeOptions is a struct to support node command
type NodeOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewNodeOptions returns initialized Options
func NewNodeOptions() *NodeOptions {
	return &NodeOptions{
		OutputOpts: output.NonTabularOutputOptions{Format: output.YAMLFormat},
	}
}

func NewNodeCmd() *cobra.Command {
	o := NewNodeOptions()
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Get the agent's node info.",
		Args:  cobra.NoArgs,
		Run:   o.runNode,
	}
	nodeCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return nodeCmd
}

// Run executes node command
func (o *NodeOptions) runNode(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	response, err := util.GetAPIClientV2(cmd).Agent().Node(ctx, &apimodels.GetAgentNodeRequest{})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not get server node: %w", err), 1)
	}

	writeErr := output.OutputOneNonTabular(cmd, o.OutputOpts, response.NodeInfo)
	if writeErr != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write node: %w", writeErr), 1)
	}
}
