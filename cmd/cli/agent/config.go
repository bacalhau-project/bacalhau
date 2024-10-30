package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

func NewConfigCmd() *cobra.Command {
	o := output.NonTabularOutputOptions{
		Format: output.YAMLFormat,
		Pretty: true,
	}
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get the agent's configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return run(cmd, api, o)
		},
	}
	configCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o))
	return configCmd
}

func run(cmd *cobra.Command, api client.API, o output.NonTabularOutputOptions) error {
	ctx := cmd.Context()
	response, err := api.Agent().Config(ctx)
	if err != nil {
		return fmt.Errorf("cannot get agent config: %w", err)
	}

	return output.OutputOneNonTabular[types.Bacalhau](cmd, o, response.Config)
}
