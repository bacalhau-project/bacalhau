package sso

import (
	"github.com/spf13/cobra"
)

func NewSSORootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sso",
		Short: "SSO Authentication commands for Bacalhau",
	}

	cmd.AddCommand(NewSSOLoginCmd())
	cmd.AddCommand(NewSSOLogoutCmd())
	cmd.AddCommand(NewSSOTokenCmd())
	return cmd
}
