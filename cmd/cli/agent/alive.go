package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
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
		Run:   o.runAlive,
	}
	aliveCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return aliveCmd
}

// Run executes alive command
func (o *AliveOptions) runAlive(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	response, err := util.GetAPIClientV2(cmd).Agent().Alive(ctx)
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not get server alive: %w", err), 1)
	}

	writeErr := output.OutputOneNonTabular(cmd, o.OutputOpts, response) //nolint
	if writeErr != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write alive: %w", writeErr), 1)
	}
}
