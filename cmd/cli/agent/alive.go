package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

// AliveOptions is a struct to support alive command
type AliveOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewAliveOptions returns initialized Options
func NewAliveOptions() *AliveOptions {
	return &AliveOptions{
		OutputOpts: output.NonTabularOutputOptions{Format: output.YAMLFormat},
	}
}

func NewAliveCmd() *cobra.Command {
	o := NewAliveOptions()
	aliveCmd := &cobra.Command{
		Use:   "alive",
		Short: "Get the agent's liveness and health info.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.runAlive(cmd, api)
		},
	}
	aliveCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return aliveCmd
}

// Run executes alive command
func (o *AliveOptions) runAlive(cmd *cobra.Command, api client.API) error {
	ctx := cmd.Context()
	response, err := api.Agent().Alive(ctx)
	if err != nil {
		return fmt.Errorf("could not get server alive: %w", err)
	}

	writeErr := output.OutputOneNonTabular(cmd, o.OutputOpts, response)
	if writeErr != nil {
		return fmt.Errorf("failed to write alive: %w", writeErr)
	}

	return nil
}
