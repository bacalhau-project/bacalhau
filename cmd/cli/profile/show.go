package profile

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// minTokenLengthForPartialRedaction is the minimum token length to show partial content
// Tokens shorter than this are fully redacted
const minTokenLengthForPartialRedaction = 8

// ShowOptions contains options for the show command
type ShowOptions struct {
	showToken bool
}

// NewShowCmd creates a new show command
func NewShowCmd() *cobra.Command {
	o := &ShowOptions{}

	cmd := &cobra.Command{
		Use:           "show [name]",
		Short:         "Show profile details.",
		Long:          `Show details for a profile. If no name is provided, shows the current profile.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       hook.ClientPreRunHooks,
		PostRunE:      hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}
			var name string
			if len(args) > 0 {
				name = args[0]
			}
			return o.run(cmd, cfg, name)
		},
	}

	cmd.Flags().BoolVar(&o.showToken, "show-token", false, "Show full token value (default: redacted)")

	return cmd
}

func (o *ShowOptions) run(cmd *cobra.Command, cfg *config.Config, name string) error {
	dataDir, ok := cfg.Get(types.DataDirKey).(string)
	if !ok {
		return fmt.Errorf("data directory configuration is invalid")
	}
	profilesDir := filepath.Join(dataDir, "profiles")
	store := profile.NewStore(profilesDir)

	// Determine which profile to show:
	// 1. If a name arg is provided, use that
	// 2. Otherwise, check context for --profile flag or BACALHAU_PROFILE env
	// 3. Otherwise, use store.GetCurrent()
	if name == "" {
		flagValue, envValue := util.GetProfileFromContext(cmd.Context())
		if flagValue != "" {
			name = flagValue
		} else if envValue != "" {
			name = envValue
		} else {
			current, err := store.GetCurrent()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}
			if current == "" {
				return fmt.Errorf("no current profile set")
			}
			name = current
		}
	}

	// Load the profile
	p, err := store.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// Check if this is the current profile
	current, _ := store.GetCurrent()
	nameDisplay := name
	if name == current {
		nameDisplay = name + " (current)"
	}

	// Format auth info
	authInfo := "none"
	if p.Auth != nil && p.Auth.Token != "" {
		tokenDisplay := redactToken(p.Auth.Token)
		if o.showToken {
			tokenDisplay = p.Auth.Token
		}
		authInfo = fmt.Sprintf("token (%s)", tokenDisplay)
	}

	// Format TLS info
	tlsInfo := "secure"
	if p.IsInsecure() {
		tlsInfo = "insecure"
	}

	// Format timeout
	timeout := p.GetTimeout()

	// Output profile details
	cmd.Printf("Name:        %s\n", nameDisplay)
	cmd.Printf("Endpoint:    %s\n", p.Endpoint)
	cmd.Printf("Auth:        %s\n", authInfo)
	cmd.Printf("TLS:         %s\n", tlsInfo)
	cmd.Printf("Timeout:     %s\n", timeout)
	if p.Description != "" {
		cmd.Printf("Description: %s\n", p.Description)
	}

	return nil
}

// redactToken redacts a token for display, showing only first and last 4 characters
func redactToken(token string) string {
	if len(token) <= minTokenLengthForPartialRedaction {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
