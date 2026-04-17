package sso

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
)

// LogoutOptions is a struct to support SSO logout functionality
type LogoutOptions struct {
	Force bool // Skip confirmation prompt
}

// NewLogoutOptions returns initialized LogoutOptions
func NewLogoutOptions() *LogoutOptions {
	return &LogoutOptions{
		Force: false,
	}
}

func NewSSOLogoutCmd() *cobra.Command {
	o := NewLogoutOptions()
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from current SSO session",
		Long:  `Logout from the current SSO session and remove stored credentials.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return bacerrors.New("failed to setup bacalhau repository config")
			}
			return o.runSSOLogout(cmd, cfg)
		},
	}

	// Add force flag to skip confirmation
	logoutCmd.Flags().BoolVarP(&o.Force, "force", "f", false, "Skip confirmation prompt")

	return logoutCmd
}

// runSSOLogout handles the SSO logout process
func (o *LogoutOptions) runSSOLogout(cmd *cobra.Command, cfg types.Bacalhau) error {
	apiURL, _ := util.ConstructAPIEndpoint(cfg.API)

	if !o.Force {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to logout from %s? (y/N): ", apiURL)
		var response string

		_, err := fmt.Fscanln(cmd.InOrStdin(), &response)
		// Ignore EOF and unexpected newline errors which happen when user just presses Enter
		if err != nil && err.Error() != "EOF" && err.Error() != "unexpected newline" {
			return err
		}
		if response != "y" && response != "Y" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logout cancelled")
			return nil
		}
	}

	authTokenPath, err := cfg.JWTTokensPath()
	if err != nil {
		return bacerrors.New("unable to find local SSO session file")
	}

	if err = util.WriteToken(authTokenPath, apiURL, nil); err != nil {
		return bacerrors.New("unable to delete SSO session credentials")
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nSuccessfully logged out from %s\n", apiURL)
	return nil
}
