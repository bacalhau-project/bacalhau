package agent

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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
			return oV.runVersion(cmd, api)
		},
	}
	versionCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&oV.OutputOpts))
	return versionCmd
}

// Run executes version command
func (oV *VersionOptions) runVersion(cmd *cobra.Command, api client.API) error {
	ctx := cmd.Context()
	serverVersionResponse, err := api.Agent().Version(ctx)
	if err != nil {
		return fmt.Errorf("could not get server version: %w", err)
	}

	v := serverVersionResponse.BuildVersionInfo
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
		return fmt.Errorf("failed to write version: %w", writeErr)
	}

	return nil
}
