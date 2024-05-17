package node

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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
			return o.runDescribe(cmd, args, api)
		},
	}
	nodeCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return nodeCmd
}

// Run executes node command
func (o *DescribeOptions) runDescribe(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	nodeID := args[0]
	response, err := api.Nodes().Get(ctx, &apimodels.GetNodeRequest{
		NodeID: nodeID,
	})
	if err != nil {
		return fmt.Errorf("could not get node %s: %w", nodeID, err)
	}

	if err = output.OutputOneNonTabular(cmd, o.OutputOpts, response.Node); err != nil {
		return fmt.Errorf("failed to write node %s: %w", nodeID, err)
	}

	return nil
}
