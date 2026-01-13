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

// SaveOptions contains options for the save command
type SaveOptions struct {
	endpoint    string
	description string
	timeout     string
	insecure    bool
	selectAfter bool
}

// NewSaveCmd creates a new save command
func NewSaveCmd() *cobra.Command {
	o := &SaveOptions{}

	cmd := &cobra.Command{
		Use:           "save <name>",
		Short:         "Create or update a profile.",
		Long:          `Create a new profile or update an existing one with connection settings.`,
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

	cmd.Flags().StringVar(&o.endpoint, "endpoint", "", "API endpoint (host:port or full URL)")
	cmd.Flags().StringVar(&o.description, "description", "", "Profile description")
	cmd.Flags().StringVar(&o.timeout, "timeout", "", "Request timeout (e.g., 30s, 1m)")
	cmd.Flags().BoolVar(&o.insecure, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().BoolVar(&o.selectAfter, "select", false, "Set as current profile after saving")

	return cmd
}

func (o *SaveOptions) run(cmd *cobra.Command, cfg *config.Config, name string) error {
	dataDir, ok := cfg.Get(types.DataDirKey).(string)
	if !ok {
		return fmt.Errorf("data directory configuration is invalid")
	}
	profilesDir := filepath.Join(dataDir, "profiles")
	store := profile.NewStore(profilesDir)

	// Load existing profile or create new
	var p *profile.Profile
	if store.Exists(name) {
		existing, err := store.Load(name)
		if err != nil {
			return fmt.Errorf("failed to load existing profile: %w", err)
		}
		p = existing
	} else {
		// New profile requires endpoint
		if o.endpoint == "" {
			return fmt.Errorf("endpoint is required when creating a new profile")
		}
		p = &profile.Profile{}
	}

	// Apply provided options
	if o.endpoint != "" {
		p.Endpoint = o.endpoint
	}
	if o.description != "" {
		p.Description = o.description
	}
	if o.timeout != "" {
		p.Timeout = o.timeout
	}
	if o.insecure {
		if p.TLS == nil {
			p.TLS = &profile.TLSConfig{}
		}
		p.TLS.Insecure = true
	}

	// Save profile (validates internally)
	if err := store.Save(name, p); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	cmd.Printf("Profile %q saved\n", name)

	// Select if requested
	if o.selectAfter {
		if err := store.SetCurrent(name); err != nil {
			return fmt.Errorf("failed to set current profile: %w", err)
		}
		cmd.Printf("Switched to profile %q\n", name)
	}

	return nil
}
