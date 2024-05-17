package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return o.runNode(cmd, api)
		},
	}
	nodeCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return nodeCmd
}

// Run executes node command
func (o *NodeOptions) runNode(cmd *cobra.Command, api client.API) error {
	ctx := cmd.Context()
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
