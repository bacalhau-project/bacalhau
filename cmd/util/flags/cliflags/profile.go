package cliflags

import "github.com/spf13/cobra"

// RegisterProfileFlag adds the --profile flag to client commands that connect to a server API.
// This flag allows users to select which profile to use for the API connection.
// Commands that don't need API connections (serve, config, devstack, profile) should NOT call this.
func RegisterProfileFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String("profile", "", "Use a specific profile for this command")
}
