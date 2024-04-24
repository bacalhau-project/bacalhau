package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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

func NewNodeCmd(cfg *config.Config) *cobra.Command {
	o := NewNodeOptions()
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Get the agent's node info.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.runNode(cmd, cfg)
		},
	}
	nodeCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return nodeCmd
}

// Run executes node command
func (o *NodeOptions) runNode(cmd *cobra.Command, cfg *config.Config) error {
	ctx := cmd.Context()
	api, err := util.GetAPIClientV2(cmd, cfg)
	if err != nil {
		return err
	}
	response, err := api.Agent().Node(ctx, &apimodels.GetAgentNodeRequest{})
	if err != nil {
		return fmt.Errorf("could not get server node: %w", err)
	}

	writeErr := output.OutputOneNonTabular(cmd, o.OutputOpts, response.NodeState)
	if writeErr != nil {
		return fmt.Errorf("failed to write node: %w", writeErr)
	}

	return nil
}
