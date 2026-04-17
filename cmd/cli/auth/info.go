package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/common"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const errorInfoHint = "Set LOG_LEVEL=DEBUG and re-run for detailed logs"

// InfoOptions is a struct to support info command
type InfoOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewInfoOptions returns initialized Options
func NewInfoOptions() *InfoOptions {
	return &InfoOptions{
		OutputOpts: output.NonTabularOutputOptions{Format: output.YAMLFormat},
	}
}

func NewInfoCmd() *cobra.Command {
	o := NewInfoOptions()
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Display authentication information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				log.Debug().Err(err).Msg("failed to setup bacalhau repository config")
				return bacerrors.New("failed to setup bacalhau repository config").
					WithHint(errorInfoHint)
			}

			api, err := util.NewAPIClientManager(cmd, cfg).GetUnauthenticatedAPIClient()
			if err != nil {
				log.Debug().Err(err).Msg("failed to initialize API client")
				return bacerrors.New("failed to fetch supported authentication details from server").
					WithHint(errorInfoHint)
			}
			return o.runInfo(cmd, api, cfg)
		},
	}
	infoCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return infoCmd
}

// Run executes info command
func (o *InfoOptions) runInfo(cmd *cobra.Command, api client.API, cfg types.Bacalhau) error {
	ctx := cmd.Context()

	currentAPIEndpoint, _ := util.ConstructAPIEndpoint(cfg.API)

	// Check environment variables
	apiKey := os.Getenv(common.BacalhauAPIKey)
	username := os.Getenv(common.BacalhauAPIUsername)
	password := os.Getenv(common.BacalhauAPIPassword)

	outputBuilder := strings.Builder{}

	// Print target environment
	outputBuilder.WriteString(fmt.Sprintf("\nTarget environment: %s\n\n", currentAPIEndpoint))

	// Print environment variable status
	outputBuilder.WriteString("Environment Variables:\n")
	outputBuilder.WriteString(fmt.Sprintf("API Key: %s\n", getEnvStatus(apiKey)))
	outputBuilder.WriteString(fmt.Sprintf("Username: %s\n", getEnvStatus(username)))
	outputBuilder.WriteString(fmt.Sprintf("Password: %s\n\n", getEnvStatus(password)))

	// Get auth config
	nodeAuthConfig, err := api.Agent().NodeAuthConfig(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("An error occurred while getting node auth config")
		outputBuilder.WriteString("Server does not support Basic Auth, API Keys, or SSO logins\n")
		_, writeErr := cmd.OutOrStdout().Write([]byte(outputBuilder.String()))
		if writeErr != nil {
			return bacerrors.New("failed to write authentication information").
				WithHint(errorInfoHint)
		}
		return nil
	}

	// Print auth config information
	outputBuilder.WriteString("Node SSO Authentication:\n")
	if nodeAuthConfig.Config.ProviderID != "" {
		outputBuilder.WriteString(fmt.Sprintf("Provider Name: %s\n", nodeAuthConfig.Config.ProviderName))
		outputBuilder.WriteString(fmt.Sprintf("Provider ID: %s\n", nodeAuthConfig.Config.ProviderID))
		outputBuilder.WriteString(fmt.Sprintf("Version: %s\n", nodeAuthConfig.Version))
	} else {
		outputBuilder.WriteString("Server does not support SSO login\n")
	}

	// Add note about environment variable precedence
	outputBuilder.WriteString("\nNote: Environment variables take precedence over other authentication mechanisms including SSO.\n")
	outputBuilder.WriteString("To use SSO login, please unset Auth related environment variables first.\n")

	_, writeErr := cmd.OutOrStdout().Write([]byte(outputBuilder.String()))
	if writeErr != nil {
		return bacerrors.New("failed to write authentication information").
			WithHint(errorInfoHint)
	}

	return nil
}

// Helper function to format environment variable status
func getEnvStatus(value string) string {
	if value != "" {
		return "Set"
	}
	return "Not Set"
}
