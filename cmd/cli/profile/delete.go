package profile

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// DeleteOptions contains options for the delete command
type DeleteOptions struct {
	force bool
}

// NewDeleteCmd creates a new delete command
func NewDeleteCmd() *cobra.Command {
	o := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:           "delete <name>",
		Short:         "Delete a profile.",
		Long:          `Delete a profile from the CLI configuration.`,
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
			return o.run(cmd, cfg, args[0])
		},
	}

	cmd.Flags().BoolVarP(&o.force, "force", "f", false, "Skip confirmation for current profile")

	return cmd
}

func (o *DeleteOptions) run(cmd *cobra.Command, cfg *config.Config, name string) error {
	dataDir, ok := cfg.Get(types.DataDirKey).(string)
	if !ok {
		return fmt.Errorf("data directory configuration is invalid")
	}
	profilesDir := filepath.Join(dataDir, "profiles")
	store := profile.NewStore(profilesDir)

	// Check if profile exists
	if !store.Exists(name) {
		return fmt.Errorf("profile %q not found", name)
	}

	// Check if this is the current profile
	current, _ := store.GetCurrent()
	if current == name && !o.force {
		// Prompt for confirmation
		cmd.Printf("Profile %q is the current profile. Delete anyway? [y/N]: ", name)

		reader := bufio.NewReader(cmd.InOrStdin())
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			cmd.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete the profile
	if err := store.Delete(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	cmd.Printf("Profile %q deleted\n", name)
	return nil
}
