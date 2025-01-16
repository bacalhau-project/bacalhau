package license

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

// AgentLicenseInspectOptions is a struct to support license command
type AgentLicenseInspectOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewAgentLicenseInspectOptions returns initialized Options
func NewAgentLicenseInspectOptions() *AgentLicenseInspectOptions {
	return &AgentLicenseInspectOptions{
		OutputOpts: output.NonTabularOutputOptions{},
	}
}

func NewAgentLicenseInspectCmd() *cobra.Command {
	o := NewAgentLicenseInspectOptions()
	licenseCmd := &cobra.Command{
		Use:   "inspect",
		Short: "Get the agent license information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.runAgentLicense(cmd, api)
		},
	}
	licenseCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return licenseCmd
}

// Run executes license command
func (o *AgentLicenseInspectOptions) runAgentLicense(cmd *cobra.Command, api client.API) error {
	ctx := cmd.Context()
	response, err := api.Agent().License(ctx)
	if err != nil {
		return fmt.Errorf("could not get agent license: %w", err)
	}

	// For JSON/YAML output
	if o.OutputOpts.Format == output.JSONFormat || o.OutputOpts.Format == output.YAMLFormat {
		return output.OutputOneNonTabular(cmd, o.OutputOpts, response.LicenseClaims)
	}

	// Create header data pairs for key-value output
	headerData := []collections.Pair[string, any]{
		{Left: "Product", Right: response.Product},
		{Left: "License ID", Right: response.LicenseID},
		{Left: "Customer ID", Right: response.CustomerID},
		{Left: "Valid Until", Right: response.ExpiresAt.Format(time.DateOnly)},
		{Left: "Version", Right: response.LicenseVersion},
	}

	// Always show Capabilities
	capabilitiesStr := "{}"
	if len(response.Capabilities) > 0 {
		var caps []string
		for k, v := range response.Capabilities {
			caps = append(caps, fmt.Sprintf("%s=%s", k, v))
		}
		capabilitiesStr = strings.Join(caps, ", ")
	}
	headerData = append(headerData, collections.Pair[string, any]{
		Left:  "Capabilities",
		Right: capabilitiesStr,
	})

	// Always show Metadata
	metadataStr := "{}"
	if len(response.Metadata) > 0 {
		var meta []string
		for k, v := range response.Metadata {
			meta = append(meta, fmt.Sprintf("%s=%s", k, v))
		}
		metadataStr = strings.Join(meta, ", ")
	}
	headerData = append(headerData, collections.Pair[string, any]{
		Left:  "Metadata",
		Right: metadataStr,
	})

	output.KeyValue(cmd, headerData)
	return nil
}
