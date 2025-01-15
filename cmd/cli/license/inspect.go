package license

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	licensepkg "github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

type InspectOptions struct {
	LicenseFile string
	OutputOpts  output.OutputOptions
}

func NewInspectOptions() *InspectOptions {
	return &InspectOptions{
		OutputOpts: output.OutputOptions{Format: output.TableFormat},
	}
}

// Add this struct after the LicenseInfo struct
type licenseFile struct {
	License string `json:"license"`
}

func NewInspectCmd() *cobra.Command {
	o := NewInspectOptions()
	cmd := &cobra.Command{
		Use:           "inspect [path]",
		Short:         "Inspect license information",
		Args:          cobra.ExactArgs(1),
		PreRun:        hook.ApplyPorcelainLogLevel,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the license file path from args
			o.LicenseFile = args[0]

			// Check if license file path is empty or just whitespace
			if len(strings.TrimSpace(o.LicenseFile)) == 0 {
				return fmt.Errorf("license file path cannot be empty")
			}

			// Check if license file exists
			if _, err := os.Stat(o.LicenseFile); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", o.LicenseFile)
			}
			return o.Run(cmd.Context(), cmd)
		},
	}

	// Add output format flags only
	cmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOpts))

	return cmd
}

func (o *InspectOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	// Read the license file
	data, err := os.ReadFile(o.LicenseFile)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	// Parse the license file
	var license licenseFile
	if err := json.Unmarshal(data, &license); err != nil {
		return fmt.Errorf("failed to parse license file: %w", err)
	}

	// Create offline license validator
	validator, err := licensepkg.NewOfflineLicenseValidator()
	if err != nil {
		return fmt.Errorf("failed to create license validator: %w", err)
	}

	// Validate the license token
	claims, err := validator.ValidateToken(license.License)
	if err != nil {
		return fmt.Errorf("invalid license: %w", err)
	}

	// For JSON/YAML output
	if o.OutputOpts.Format == output.JSONFormat || o.OutputOpts.Format == output.YAMLFormat {
		return output.OutputOne(cmd, nil, o.OutputOpts, claims)
	}

	// Create header data pairs for key-value output
	headerData := []collections.Pair[string, any]{
		{Left: "Product", Right: claims.Product},
		{Left: "License ID", Right: claims.LicenseID},
		{Left: "Customer ID", Right: claims.CustomerID},
		{Left: "Valid Until", Right: claims.ExpiresAt.Format(time.DateOnly)},
		{Left: "Version", Right: claims.LicenseVersion},
	}

	// Always show Capabilities
	capabilitiesStr := "{}"
	if len(claims.Capabilities) > 0 {
		var caps []string
		for k, v := range claims.Capabilities {
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
	if len(claims.Metadata) > 0 {
		var meta []string
		for k, v := range claims.Metadata {
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
