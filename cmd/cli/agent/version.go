package agent

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/spf13/cobra"
)

// VersionOptions is a struct to support version command
type VersionOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewVersionOptions returns initialized Options
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		OutputOpts: output.NonTabularOutputOptions{},
	}
}

func NewVersionCmd() *cobra.Command {
	oV := NewVersionOptions()
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Get the agent version.",
		Args:  cobra.NoArgs,
		Run:   oV.runVersion,
	}
	versionCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&oV.OutputOpts))
	return versionCmd
}

// Run executes version command
func (oV *VersionOptions) runVersion(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	serverVersionResponse, err := util.GetAPIClientV2(ctx).Agent().Version()
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not get server version: %w", err), 1)
	}

	v := serverVersionResponse.Version
	var writeErr error

	// default output if no format is specified
	if oV.OutputOpts.Format == "" {
		outputBuilder := strings.Builder{}
		outputBuilder.WriteString(fmt.Sprintf("Bacalhau %s\n", v.GitVersion))
		outputBuilder.WriteString(fmt.Sprintf("BuildDate %s\n", v.BuildDate))
		outputBuilder.WriteString(fmt.Sprintf("GitCommit %s\n", v.GitCommit))
		_, writeErr = cmd.OutOrStdout().Write([]byte(outputBuilder.String()))
	} else {
		writeErr = output.OutputOneNonTabular(cmd, oV.OutputOpts, v)
	}

	if writeErr != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write version: %w", writeErr), 1)
	}
}
