package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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

func NewAliveCmd(cfg *config.Config) *cobra.Command {
	o := NewAliveOptions()
	aliveCmd := &cobra.Command{
		Use:   "alive",
		Short: "Get the agent's liveness and health info.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return o.runAlive(cmd, cfg)
		},
	}
	aliveCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return aliveCmd
}

// Run executes alive command
func (o *AliveOptions) runAlive(cmd *cobra.Command, cfg *config.Config) error {
	ctx := cmd.Context()
	api, err := util.GetAPIClientV2(cmd, cfg)
	if err != nil {
		return err
	}
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
