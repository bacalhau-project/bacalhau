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

// NewSelectCmd creates a new select command
func NewSelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "select <name>",
		Short:         "Set a profile as current.",
		Long:          `Set a profile as the current active profile for CLI operations.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       hook.ClientPreRunHooks,
		PostRunE:      hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}
			return runSelect(cmd, cfg, args[0])
		},
	}

	return cmd
}

func runSelect(cmd *cobra.Command, cfg *config.Config, name string) error {
	dataDir, ok := cfg.Get(types.DataDirKey).(string)
	if !ok {
		return fmt.Errorf("data directory configuration is invalid")
	}
	profilesDir := filepath.Join(dataDir, "profiles")
	store := profile.NewStore(profilesDir)

	// SetCurrent validates that the profile exists
	if err := store.SetCurrent(name); err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}

	cmd.Printf("Switched to profile %q\n", name)
	return nil
}
