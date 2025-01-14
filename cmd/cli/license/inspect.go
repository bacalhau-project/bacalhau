package license

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
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

func NewInspectCmd() *cobra.Command {
	o := NewInspectOptions()
	cmd := &cobra.Command{
		Use:           "inspect",
		Short:         "Inspect license information",
		Args:          cobra.NoArgs,
		PreRun:        hook.ApplyPorcelainLogLevel,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Check if license file path is empty or just whitespace
			if o.LicenseFile == "" || len(strings.TrimSpace(o.LicenseFile)) == 0 {
				return fmt.Errorf("required flag \"license-file\" not set")
			}

			// Check if license file exists
			if _, err := os.Stat(o.LicenseFile); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", o.LicenseFile)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return o.Run(cmd.Context(), cmd)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&o.LicenseFile, "license-file", "", "Path to the license file")
	cmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOpts))

	// Set required flag
	_ = cmd.MarkFlagRequired("license-file")

	return cmd
}

// LicenseInfo represents the structure of license information
type LicenseInfo struct {
	Product        string
	LicenseID      string
	CustomerID     string
	ValidUntil     string
	LicenseVersion string
	Capabilities   map[string]string
}

// jsonOutput is used for JSON/YAML serialization to include metadata
type jsonOutput struct {
	LicenseInfo
	Metadata map[string]string
}

// MarshalJSON implements custom JSON marshaling
func (j jsonOutput) MarshalJSON() ([]byte, error) {
	type Alias jsonOutput // prevent recursion
	data := struct {
		Alias
		Metadata map[string]string `json:"Metadata"`
	}{
		Alias:    Alias(j),
		Metadata: j.Metadata,
	}
	return json.Marshal(data)
}

// MarshalYAML implements custom YAML marshaling
func (j jsonOutput) MarshalYAML() (interface{}, error) {
	type Alias jsonOutput // prevent recursion
	data := struct {
		Alias
		Metadata map[string]string `yaml:"Metadata"`
	}{
		Alias:    Alias(j),
		Metadata: j.Metadata,
	}
	return data, nil
}

// Add this struct after the LicenseInfo struct
type licenseFile struct {
	License string `json:"license"`
}

// Update table columns
var licenseProductColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "PRODUCT"},
	Value:        func(l LicenseInfo) string { return l.Product },
}

var licenseLicenseIDColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "LICENSE ID"},
	Value:        func(l LicenseInfo) string { return l.LicenseID },
}

var licenseCustomerIDColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "CUSTOMER ID"},
	Value:        func(l LicenseInfo) string { return l.CustomerID },
}

var licenseValidUntilColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "VALID UNTIL"},
	Value:        func(l LicenseInfo) string { return l.ValidUntil },
}

var licenseLicenseVersionColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "VERSION"},
	Value:        func(l LicenseInfo) string { return l.LicenseVersion },
}

var licenseCapabilitiesColumn = output.TableColumn[LicenseInfo]{
	ColumnConfig: table.ColumnConfig{Name: "CAPABILITIES"},
	Value: func(l LicenseInfo) string {
		if len(l.Capabilities) == 0 {
			return "none"
		}
		var caps []string
		for k, v := range l.Capabilities {
			caps = append(caps, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(caps, ", ")
	},
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

	// Create license info from the validated claims
	licenseInfo := LicenseInfo{
		Product:        claims.Product,
		LicenseID:      claims.LicenseID,
		CustomerID:     claims.CustomerID,
		ValidUntil:     claims.ExpiresAt.Format(time.DateOnly),
		LicenseVersion: claims.LicenseVersion,
		Capabilities:   claims.Capabilities,
	}

	// For JSON/YAML output, wrap the license info with metadata
	if o.OutputOpts.Format == output.JSONFormat || o.OutputOpts.Format == output.YAMLFormat {
		return output.OutputOne(cmd, nil, o.OutputOpts, jsonOutput{
			LicenseInfo: licenseInfo,
			Metadata:    claims.Metadata,
		})
	}

	columns := []output.TableColumn[LicenseInfo]{
		licenseProductColumn,
		licenseLicenseIDColumn,
		licenseCustomerIDColumn,
		licenseValidUntilColumn,
		licenseLicenseVersionColumn,
		licenseCapabilitiesColumn,
	}

	return output.OutputOne(cmd, columns, o.OutputOpts, licenseInfo)
}
